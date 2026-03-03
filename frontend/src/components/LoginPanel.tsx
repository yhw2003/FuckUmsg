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
      <form
        className="login-form"
        onSubmit={(event) => {
          event.preventDefault()
          if (!password || loading) {
            return
          }
          void onLogin()
        }}
      >
        <input
          className="input"
          type="password"
          placeholder="输入访问口令"
          aria-label="访问口令"
          value={password}
          onChange={(event) => onPasswordChange(event.target.value)}
        />

        <button className="button button-primary login-submit" type="submit" disabled={loading || !password}>
          {loading ? '登录中...' : '进入看板'}
        </button>
      </form>

      {loginError ? <div className="alert-error">{loginError}</div> : null}
    </>
  )
}

export default LoginPanel
