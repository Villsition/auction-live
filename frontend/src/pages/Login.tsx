import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useAuth } from '../store/AuthContext';

export default function Login() {
  const { login } = useAuth();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [showForceModal, setShowForceModal] = useState(false);

  const doLogin = async (force = false) => {
    setError('');
    setLoading(true);
    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password, force }),
      });
      const json = await res.json();
      if (json.code === 40901 && !force) {
        setShowForceModal(true);
        setLoading(false);
        return;
      }
      if (json.code !== 0) throw new Error(json.message);
      localStorage.setItem('token', json.data.token);
      localStorage.setItem('user', JSON.stringify(json.data.user));
      window.location.href = '/';
    } catch (err: any) {
      setError(err.message);
    }
    setLoading(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    await doLogin(false);
  };

  return (
    <div style={{
      minHeight: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center',
      background: 'linear-gradient(135deg, #0a0a1a 0%, #1a1040 30%, #0d1b2a 60%, #0a0a1a 100%)',
      position: 'relative', overflow: 'hidden', fontFamily: "'Noto Sans SC', 'PingFang SC', system-ui, sans-serif",
    }}>
      {/* Ambient light orbs */}
      <div style={{
        position: 'absolute', top: '-15%', right: '-10%', width: '60vw', height: '60vw',
        background: 'radial-gradient(circle, rgba(99,102,241,0.12) 0%, transparent 70%)',
        borderRadius: '50%', pointerEvents: 'none',
      }} />
      <div style={{
        position: 'absolute', bottom: '-20%', left: '-5%', width: '50vw', height: '50vw',
        background: 'radial-gradient(circle, rgba(56,189,248,0.08) 0%, transparent 70%)',
        borderRadius: '50%', pointerEvents: 'none',
      }} />
      <div style={{
        position: 'absolute', top: '40%', left: '50%', transform: 'translate(-50%, -50%)',
        width: '40vw', height: '40vw',
        background: 'radial-gradient(circle, rgba(139,92,246,0.06) 0%, transparent 70%)',
        borderRadius: '50%', pointerEvents: 'none',
      }} />

      {/* Glass card */}
      <div style={{
        width: 420, padding: '48px 40px 40px', borderRadius: 24,
        background: 'rgba(255,255,255,0.04)',
        backdropFilter: 'blur(24px)',
        WebkitBackdropFilter: 'blur(24px)',
        border: '1px solid rgba(255,255,255,0.08)',
        boxShadow: '0 24px 80px rgba(0,0,0,0.4), 0 0 1px rgba(255,255,255,0.06) inset',
        position: 'relative', zIndex: 1,
        animation: 'fadeIn 0.6s ease-out',
      }}>
        {/* Top accent line */}
        <div style={{
          position: 'absolute', top: 0, left: 60, right: 60, height: 1,
          background: 'linear-gradient(90deg, transparent, rgba(99,102,241,0.4), rgba(56,189,248,0.4), transparent)',
        }} />

        {/* Logo / Title */}
        <div style={{
          textAlign: 'center', marginBottom: 36,
          animation: 'slideDown 0.5s ease-out 0.1s both',
        }}>
          <div style={{
            display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
            width: 56, height: 56, borderRadius: 16, marginBottom: 16,
            background: 'linear-gradient(135deg, rgba(99,102,241,0.2), rgba(56,189,248,0.15))',
            border: '1px solid rgba(99,102,241,0.2)',
          }}>
            <span style={{ fontSize: 26 }}>🔨</span>
          </div>
          <h1 style={{
            margin: 0, fontSize: 26, fontWeight: 700, letterSpacing: 1,
            background: 'linear-gradient(135deg, #e2e8f0 0%, #a5b4fc 50%, #7dd3fc 100%)',
            WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent',
          }}>竞拍直播间</h1>
          <p style={{ margin: '8px 0 0', fontSize: 13, color: 'rgba(148,163,184,0.6)', letterSpacing: 0.5 }}>
            实时竞拍 · 一秒千金
          </p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 18 }}>
            {/* Username */}
            <div style={{
              animation: 'slideUp 0.5s ease-out 0.2s both',
            }}>
              <label style={{
                display: 'block', fontSize: 12, color: 'rgba(148,163,184,0.7)',
                marginBottom: 6, letterSpacing: 1, textTransform: 'uppercase', fontWeight: 500,
              }}>账号</label>
              <div style={{
                display: 'flex', alignItems: 'center', gap: 10, padding: '0 16px',
                height: 48, borderRadius: 12,
                background: 'rgba(255,255,255,0.03)',
                border: '1px solid rgba(255,255,255,0.08)',
                transition: 'border-color 0.2s, box-shadow 0.2s',
              }}
                onFocusCapture={e => {
                  const el = e.currentTarget;
                  el.style.borderColor = 'rgba(99,102,241,0.4)';
                  el.style.boxShadow = '0 0 0 3px rgba(99,102,241,0.08)';
                }}
                onBlurCapture={e => {
                  const el = e.currentTarget;
                  el.style.borderColor = 'rgba(255,255,255,0.08)';
                  el.style.boxShadow = 'none';
                }}
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="rgba(148,163,184,0.4)" strokeWidth="2" strokeLinecap="round">
                  <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
                </svg>
                <input
                  placeholder="请输入用户名"
                  value={username}
                  onChange={e => setUsername(e.target.value)}
                  required
                  style={{
                    flex: 1, height: '100%', background: 'none', border: 'none',
                    outline: 'none', fontSize: 15, color: '#e2e8f0',
                    fontFamily: 'inherit',
                  }}
                />
              </div>
            </div>

            {/* Password */}
            <div style={{
              animation: 'slideUp 0.5s ease-out 0.3s both',
            }}>
              <label style={{
                display: 'block', fontSize: 12, color: 'rgba(148,163,184,0.7)',
                marginBottom: 6, letterSpacing: 1, textTransform: 'uppercase', fontWeight: 500,
              }}>密码</label>
              <div style={{
                display: 'flex', alignItems: 'center', gap: 10, padding: '0 16px',
                height: 48, borderRadius: 12,
                background: 'rgba(255,255,255,0.03)',
                border: '1px solid rgba(255,255,255,0.08)',
                transition: 'border-color 0.2s, box-shadow 0.2s',
              }}
                onFocusCapture={e => {
                  const el = e.currentTarget;
                  el.style.borderColor = 'rgba(99,102,241,0.4)';
                  el.style.boxShadow = '0 0 0 3px rgba(99,102,241,0.08)';
                }}
                onBlurCapture={e => {
                  const el = e.currentTarget;
                  el.style.borderColor = 'rgba(255,255,255,0.08)';
                  el.style.boxShadow = 'none';
                }}
              >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="rgba(148,163,184,0.4)" strokeWidth="2" strokeLinecap="round">
                  <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/>
                </svg>
                <input
                  type="password"
                  placeholder="请输入密码"
                  value={password}
                  onChange={e => setPassword(e.target.value)}
                  required
                  style={{
                    flex: 1, height: '100%', background: 'none', border: 'none',
                    outline: 'none', fontSize: 15, color: '#e2e8f0',
                    fontFamily: 'inherit',
                  }}
                />
              </div>
            </div>

            {/* Error */}
            {error && (
              <div style={{
                color: '#fca5a5', fontSize: 13, textAlign: 'center',
                padding: '10px 16px', borderRadius: 8,
                background: 'rgba(239,68,68,0.1)', border: '1px solid rgba(239,68,68,0.15)',
                animation: 'shake 0.4s ease-out',
              }}>
                {error}
              </div>
            )}

            {/* Button */}
            <button
              type="submit"
              disabled={loading}
              style={{
                marginTop: 8, height: 48, borderRadius: 12, border: 'none',
                cursor: loading ? 'not-allowed' : 'pointer',
                fontSize: 15, fontWeight: 600, letterSpacing: 0.5,
                color: '#fff',
                background: loading
                  ? 'linear-gradient(135deg, rgba(99,102,241,0.4), rgba(56,189,248,0.3))'
                  : 'linear-gradient(135deg, #6366f1, #3b82f6)',
                boxShadow: loading ? 'none' : '0 4px 20px rgba(99,102,241,0.3)',
                transition: 'all 0.25s ease',
                position: 'relative', overflow: 'hidden',
                animation: 'slideUp 0.5s ease-out 0.4s both',
              }}
              onMouseEnter={e => {
                if (!loading) {
                  e.currentTarget.style.transform = 'translateY(-1px)';
                  e.currentTarget.style.boxShadow = '0 6px 28px rgba(99,102,241,0.45)';
                }
              }}
              onMouseLeave={e => {
                e.currentTarget.style.transform = 'translateY(0)';
                e.currentTarget.style.boxShadow = '0 4px 20px rgba(99,102,241,0.3)';
              }}
            >
              {/* Button shine */}
              <div style={{
                position: 'absolute', top: 0, left: 0, width: '100%', height: '100%',
                background: 'linear-gradient(90deg, transparent, rgba(255,255,255,0.08), transparent)',
                transform: 'skewX(-20deg) translateX(-100%)',
                animation: 'btnShine 3s infinite',
              }} />
              {loading ? '登录中...' : '登录'}
            </button>

            {/* Register link */}
            <div style={{
              textAlign: 'center', animation: 'slideUp 0.5s ease-out 0.5s both',
            }}>
              <span style={{ fontSize: 13, color: 'rgba(148,163,184,0.4)' }}>还没有账号？</span>
              <Link to="/register" style={{
                fontSize: 13, color: '#818cf8', textDecoration: 'none',
                marginLeft: 4, fontWeight: 500,
                transition: 'color 0.2s',
              }}
                onMouseEnter={e => (e.currentTarget.style.color = '#a5b4fc')}
                onMouseLeave={e => (e.currentTarget.style.color = '#818cf8')}
              >立即注册</Link>
            </div>
          </div>
        </form>
      </div>

      {/* Force login confirmation modal */}
      {showForceModal && (
        <div style={{position:'fixed',inset:0,zIndex:999,display:'flex',alignItems:'center',justifyContent:'center',background:'rgba(0,0,0,0.6)',backdropFilter:'blur(4px)'}}>
          <div style={{background:'rgba(15,15,40,0.98)',backdropFilter:'blur(20px)',borderRadius:16,padding:'28px 24px 20px',width:360,border:'1px solid rgba(255,255,255,0.08)',animation:'fadeIn 0.3s ease-out',position:'relative'}}>
            <button onClick={()=>setShowForceModal(false)} style={{position:'absolute',top:12,right:12,background:'none',border:'none',color:'rgba(148,163,184,0.5)',fontSize:18,cursor:'pointer'}}>✕</button>
            <div style={{fontSize:16,fontWeight:700,color:'#e2e8f0',marginBottom:8}}>当前账号已在其他设备登录</div>
            <div style={{fontSize:13,color:'rgba(148,163,184,0.6)',marginBottom:20,lineHeight:1.5}}>如果继续登录，其他设备的登录状态将失效</div>
            <button onClick={()=>{setShowForceModal(false);doLogin(true)}} style={{width:'100%',padding:12,background:'linear-gradient(135deg,#6366f1,#3b82f6)',color:'#fff',border:'none',borderRadius:10,fontSize:14,fontWeight:600,cursor:'pointer'}}>继续登录</button>
          </div>
        </div>
      )}

      {/* Animations */}
      <style>{`
        @keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
        @keyframes slideDown { from { opacity: 0; transform: translateY(-16px); } to { opacity: 1; transform: translateY(0); } }
        @keyframes slideUp { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: translateY(0); } }
        @keyframes shake { 0%,100% { transform: translateX(0); } 20% { transform: translateX(-6px); } 40% { transform: translateX(6px); } 60% { transform: translateX(-4px); } 80% { transform: translateX(4px); } }
        @keyframes btnShine { 0% { transform: skewX(-20deg) translateX(-100%); } 100% { transform: skewX(-20deg) translateX(300%); } }
      `}</style>
    </div>
  );
}
