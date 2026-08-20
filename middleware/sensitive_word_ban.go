package middleware

import (
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

const sensitiveWordBanBypassKey = "sensitive-word-ban-bypass"

func sensitiveWordBanAPIPath(path string) bool {
	return strings.HasPrefix(path, "/api") ||
		strings.HasPrefix(path, "/v1") ||
		strings.HasPrefix(path, "/v1beta") ||
		strings.HasPrefix(path, "/pg") ||
		strings.HasPrefix(path, "/mj") ||
		strings.Contains(path, "/mj/") ||
		strings.HasSuffix(path, "/mj") ||
		strings.HasPrefix(path, "/suno") ||
		strings.HasPrefix(path, "/kling") ||
		strings.HasPrefix(path, "/jimeng")
}

func SensitiveWordBanGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		switch path {
		case "/api/sensitive-word-ban/login", "/api/sensitive-word-ban/verify-2fa", "/api/sensitive-word-ban/confirm":
			c.Set(sensitiveWordBanBypassKey, true)
			c.Next()
			return
		}
		banned, err := model.IsSensitiveWordIPBanned(c.ClientIP())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"success": false, "code": "IP_BAN_CHECK_FAILED", "message": "服务暂不可用"})
			return
		}
		if !banned {
			c.Next()
			return
		}
		if sensitiveWordBanAPIPath(path) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"success": false, "code": "IP_BANNED", "message": "此 IP 已被封禁，请联系管理员"})
			return
		}
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusForbidden, "text/html; charset=utf-8", []byte(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>IP 已被封禁</title><style>html,body{margin:0;min-height:100%;background:#080808;color:#fff;font-family:system-ui,sans-serif}main{min-height:100vh;display:flex;flex-direction:column;align-items:center;justify-content:center;text-align:center;padding:24px;box-sizing:border-box}h1{font-size:clamp(3rem,10vw,8rem);margin:0 0 20px;color:#ff3b30}p{font-size:clamp(1.2rem,3vw,2rem);margin:0 0 42px}form{width:min(360px,100%);display:grid;gap:12px}input,button{font:inherit;padding:12px 14px;border-radius:6px;border:1px solid #444;box-sizing:border-box}button{cursor:pointer;background:#fff;color:#000}.hidden{display:none}</style></head><body><main><h1>此 IP 已被封禁</h1><p>请联系管理员</p><form id="login"><input name="username" autocomplete="username" placeholder="管理员账号" required><input name="password" type="password" autocomplete="current-password" placeholder="密码" required><button>管理员登录</button></form><form id="verify" class="hidden"><input name="code" inputmode="numeric" placeholder="2FA 验证码" required><button>验证并继续</button></form><button id="confirm" class="hidden">确认解封并进入站点</button><div id="message"></div></main><script>(function(){let token='';const msg=document.getElementById('message');const login=document.getElementById('login');const verify=document.getElementById('verify');const confirm=document.getElementById('confirm');async function post(path,body){const r=await fetch(path,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});return r.json()}function showError(v){msg.textContent=v.message||'操作失败'}login.addEventListener('submit',async e=>{e.preventDefault();const r=await post('/api/sensitive-word-ban/login',Object.fromEntries(new FormData(login)));if(!r.success){showError(r);return}token=r.data.confirm_token||r.data.flow_token;if(r.data.require_2fa){login.classList.add('hidden');verify.classList.remove('hidden')}else{confirm.classList.remove('hidden');login.classList.add('hidden')}});verify.addEventListener('submit',async e=>{e.preventDefault();const r=await post('/api/sensitive-word-ban/verify-2fa',{flow_token:token,code:new FormData(verify).get('code')});if(!r.success){showError(r);return}token=r.data.confirm_token;verify.classList.add('hidden');confirm.classList.remove('hidden')});confirm.addEventListener('click',async()=>{const r=await post('/api/sensitive-word-ban/confirm',{flow_token:token});if(!r.success){showError(r);return}location.reload()})})()</script></body></html>`))
		c.Abort()
	}
}
