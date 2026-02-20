interface LoginPanelProps {
  password: string
  loading: boolean
  loginError: string
  onPasswordChange: (value: string) => void
  onLogin: () => Promise<void>
}

function LoginPanel({ password, loading, loginError, onPasswordChange, onLogin }: LoginPanelProps) {
  const inputId = 'access-password'

  return (
    <>
      <div className="panel-header">
        <h2 className="panel-title">登录</h2>
      </div>
      <form
        className="login-grid"
        onSubmit={(event) => {
          event.preventDefault()
          if (!password || loading) {
            return
          }
          void onLogin()
        }}
      >
        <label className="sr-only" htmlFor={inputId}>
          访问口令
        </label>
        <input
          id={inputId}
          className="input"
          type="password"
          placeholder="输入访问口令"
          value={password}
          onChange={(event) => onPasswordChange(event.target.value)}
        />
        <button className="button" type="submit" disabled={loading || !password}>
          {loading ? '登录中...' : '进入看板'}
        </button>
      </form>
      {loginError ? <div className="error">{loginError}</div> : null}
    </>
  )
}

export default LoginPanel
