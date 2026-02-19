interface LoginPanelProps {
  password: string
  loading: boolean
  loginError: string
  onPasswordChange: (value: string) => void
  onLogin: () => Promise<void>
}

function LoginPanel({ password, loading, loginError, onPasswordChange, onLogin }: LoginPanelProps) {
  return (
    <>
      <div className="panel-header">
        <div className="panel-title">登录</div>
      </div>
      <div className="login-grid">
        <input
          className="input"
          type="password"
          placeholder="输入访问口令"
          value={password}
          onChange={(event) => onPasswordChange(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter' && password) {
              void onLogin()
            }
          }}
        />
        <button className="button" onClick={() => void onLogin()} disabled={loading || !password}>
          {loading ? '登录中...' : '进入看板'}
        </button>
      </div>
      {loginError ? <div className="error">{loginError}</div> : null}
    </>
  )
}

export default LoginPanel
