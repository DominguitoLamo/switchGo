package switchgo

import (
	"sync"
	"time"
)

const (
	HUAWEI = "huawei"
	H3C    = "h3c"
	CISCO  = "cisco"
)

var (
	HuaweiNoPage = "screen-length 0 temporary"
	H3cNoPage    = "screen-length disable"
	CiscoNoPage  = "terminal length 0"
)

/**
 * session（SSHSession）的管理类，会统一缓存打开的session，自动处理未使用超过10分钟的session
 * @attr sessionCache:缓存所有打开的map（10分钟内使用过的），sessionLocker设备锁，globalLocker全局锁
 * @author shenbowei
 */
type SessionManager struct {
	sessionCache           map[string]*SSHSession
	sessionLocker          map[string]*sync.Mutex
	sessionCacheLocker     *sync.RWMutex
	sessionLockerMapLocker *sync.RWMutex
}

/**
 * 创建一个SessionManager，相当于SessionManager的构造函数
 * @return SessionManager实例
 * @author shenbowei
 */
func NewSessionManager() *SessionManager {
	sessionManager := new(SessionManager)
	sessionManager.sessionCache = make(map[string]*SSHSession, 0)
	sessionManager.sessionLocker = make(map[string]*sync.Mutex, 0)
	sessionManager.sessionCacheLocker = new(sync.RWMutex)
	sessionManager.sessionLockerMapLocker = new(sync.RWMutex)
	//启动自动清理的线程，清理10分钟未使用的session缓存
	sessionManager.runAutoClean()
	return sessionManager
}


/**
 * 从缓存中获取session。如果不存在或者不可用，则重新创建
 * @param  user ssh连接的用户名, password 密码, ipPort 交换机的ip和端口
 * @return SSHSession
 * @author shenbowei
 */
 func (this *SessionManager) GetSSHSession(config *SSHConfig, brand string) (*SSHSession, error) {
	sessionKey := config.GetSessionKey()
	session := this.getSessionCache(sessionKey)
	if session != nil {
		//返回前要验证是否可用，不可用要重新创建并更新缓存
		if session.CheckSelf() {
			DebugLog("-----GetSession from cache-----")
			session.UpdateLastUseTime()
			return session, nil
		}
		DebugLog("Check session failed")
	}
	//如果不存在或者验证失败，需要重新连接，并更新缓存
	if err := this.updateSession(config, brand); err != nil {
		ErrorLog("SSH session pool updateSession err:%s", err.Error())
		return nil, err
	} else {
		return this.getSessionCache(sessionKey), nil
	}
}

func (this *SessionManager) getSessionCache(sessionKey string) *SSHSession {
	this.sessionCacheLocker.RLock()
	defer this.sessionCacheLocker.RUnlock()
	cache, ok := this.sessionCache[sessionKey]
	if ok {
		return cache
	} else {
		return nil
	}
}

func (this *SessionManager) setSessionCache(sessionKey string, session *SSHSession) {
	this.sessionCacheLocker.Lock()
	defer this.sessionCacheLocker.Unlock()
	this.sessionCache[sessionKey] = session
}

/**
 * 更新session缓存中的session，连接设备，打开会话，初始化会话（等待登录，识别设备类型，执行禁止分页），添加到缓存
 * @param  user ssh连接的用户名, password 密码, ipPort 交换机的ip和端口
 * @return 执行的错误
 * @author shenbowei
 */
 func (this *SessionManager) updateSession(config *SSHConfig, brand string) error {
	mySession, err := NewSSHSession(config, brand, this)
	if err != nil {
		ErrorLog("NewSSHSession err:%s", err.Error())
		return err
	}
	//初始化session，包括等待登录输出和禁用分页
	this.initSession(mySession)
	//更新session的缓存
	this.setSessionCache(config.GetSessionKey(), mySession)
	return nil
}

/**
 * 初始化会话（等待登录，识别设备类型，执行禁止分页）
 * @param  session:需要执行初始化操作的SSHSession
 * @author shenbowei
 */
 func (this *SessionManager) initSession(session *SSHSession) {
	switch session.brand {
	case HUAWEI:
		session.WriteChannel(HuaweiNoPage)
		break
	case H3C:
		session.WriteChannel(H3cNoPage)
		break
	case CISCO:
		session.WriteChannel(CiscoNoPage)
		break
	default:
		return
	}
	session.ReadChannelExpect(time.Second, "#", ">", "]")
}

/**
 * 给指定的session上锁
 * @param  sessionKey:session的索引键值
 * @author shenbowei
 */
 func (this *SessionManager) lockSession(sessionKey string) {
	this.sessionLockerMapLocker.Lock()
	defer this.sessionLockerMapLocker.Unlock()
	mutex, ok := this.sessionLocker[sessionKey]
	if !ok {
		//如果获取不到锁，需要创建锁，主要更新锁存的时候需要上全局锁
		mutex = new(sync.Mutex)
		this.sessionLocker[sessionKey] = mutex
	}
	mutex.Lock()
}

/**
 * 给指定的session解锁
 * @param  sessionKey:session的索引键值
 * @author shenbowei
 */
func (this *SessionManager) unlockSession(sessionKey string) {
	this.sessionLockerMapLocker.Lock()
	this.sessionLocker[sessionKey].Unlock()
	this.sessionLockerMapLocker.Unlock()
}

/**
 * 开始自动清理session缓存中未使用超过10分钟的session
 * @author shenbowei
 */
 func (this *SessionManager) runAutoClean() {
	go func() {
		for {
			timeoutSessionIndex := this.getTimeoutSessionIndex()
			this.sessionCacheLocker.Lock()
			for _, sessionKey := range timeoutSessionIndex {
				this.lockSession(sessionKey)
				delete(this.sessionCache, sessionKey)
				this.unlockSession(sessionKey)
			}
			this.sessionCacheLocker.Unlock()
			time.Sleep(30 * time.Second)
		}
	}()
}

/**
 * 获取所有超时（10分钟未使用）session在cache的sessionKey
 * @return []string 所有超时的sessionKey数组
 * @author shenbowei
 */
 func (this *SessionManager) getTimeoutSessionIndex() []string {
	timeoutSessionIndex := make([]string, 0)
	this.sessionCacheLocker.RLock()
	defer func() {
		this.sessionCacheLocker.RUnlock()
		if err := recover(); err != nil {
			ErrorLog("SSHSessionManager getTimeoutSessionIndex err:%s", err)
		}
	}()
	for sessionKey, sshSession := range this.sessionCache {
		timeDuratime := time.Since(sshSession.GetLastUseTime())
		if timeDuratime.Minutes() > 10 {
			DebugLog("RunAutoClean close session<%s, unuse time=%s>", sessionKey, timeDuratime.String())
			sshSession.Close()
			timeoutSessionIndex = append(timeoutSessionIndex, sessionKey)
		}
	}
	return timeoutSessionIndex
}

func (this *SessionManager) DeleteSession(sessionKey string) {
	this.sessionCacheLocker.Lock()
	defer this.sessionCacheLocker.Unlock()
	this.sessionCache[sessionKey] = nil
}
