package middleware

import (
	"net/http"
	"one-api/common/config"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	gsessions "github.com/gorilla/sessions"
)

// DynamicCookieStore 让会话 Cookie 的 Secure 标记按每个请求动态决定。
//
// gin-contrib/sessions 的 cookie.Store 把 Options 存在共享实例上，启动时定死，
// 无法同时满足「公网 HTTPS 反代加 Secure」和「内网 HTTP 直连不加 Secure」。
// 这里为每次请求构造一个使用相同密钥的独立 store，密钥不变，Cookie 仍可互通，
// 但 Secure / MaxAge 可以读取后台即时修改的配置。
type DynamicCookieStore struct {
	authKey  []byte
	blockKey []byte
	base     sessions.Options
}

func NewDynamicCookieStore(authKey, blockKey []byte) *DynamicCookieStore {
	return &DynamicCookieStore{
		authKey:  authKey,
		blockKey: blockKey,
		base: sessions.Options{
			Path:     "/",
			MaxAge:   config.CookieMaxAge(),
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
		},
	}
}

// Options 实现 sessions.Store 接口；Secure 会在每次请求时被覆盖。
func (s *DynamicCookieStore) Options(options sessions.Options) {
	s.base = options
}

func (s *DynamicCookieStore) storeFor(r *http.Request) cookie.Store {
	store := cookie.NewStore(s.authKey, s.blockKey)
	options := s.base
	options.MaxAge = config.CookieMaxAge()
	options.Secure = config.CookieSecureForRequest(r)
	store.Options(options)
	return store
}

func (s *DynamicCookieStore) Get(r *http.Request, name string) (*gsessions.Session, error) {
	return s.storeFor(r).Get(r, name)
}

func (s *DynamicCookieStore) New(r *http.Request, name string) (*gsessions.Session, error) {
	return s.storeFor(r).New(r, name)
}

func (s *DynamicCookieStore) Save(r *http.Request, w http.ResponseWriter, session *gsessions.Session) error {
	return s.storeFor(r).Save(r, w, session)
}
