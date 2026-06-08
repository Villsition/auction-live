import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../store/AuthContext';

export default function Register() {
  const { register } = useAuth();
  const navigate = useNavigate();
  const [form, setForm] = useState({ username: '', password: '', nickname: '', role: 0 });
  const [avatar, setAvatar] = useState<File | null>(null);
  const [avatarPreview, setAvatarPreview] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [showRoleMenu, setShowRoleMenu] = useState(false);
  const roleLabels = ['买家', '卖家'];

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (file) {
      setAvatar(file);
      const reader = new FileReader();
      reader.onload = () => setAvatarPreview(reader.result as string);
      reader.readAsDataURL(file);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      if (avatar) {
        // Use FormData for file upload
        const fd = new FormData();
        fd.append('username', form.username);
        fd.append('password', form.password);
        fd.append('nickname', form.nickname);
        fd.append('role', String(form.role));
        fd.append('avatar', avatar);
        const res = await fetch('/api/auth/register', { method: 'POST', body: fd });
        const json = await res.json();
        if (json.code !== 0) throw new Error(json.message);
        // Save token and user manually (since we bypassed the auth context)
        localStorage.setItem('token', json.data.token);
        localStorage.setItem('user', JSON.stringify(json.data.user));
      } else {
        await register(form.username, form.password, form.nickname, form.role);
      }
      navigate('/', { replace: true });
    } catch (err: any) { setError(err.message); }
    setLoading(false);
  };

  const inputWrapperStyle = (delay: number): React.CSSProperties => ({
    animation: `slideUp 0.5s ease-out ${delay}s both`,
  });

  const labelStyle: React.CSSProperties = {
    display: 'block', fontSize: 12, color: 'rgba(148,163,184,0.7)',
    marginBottom: 6, letterSpacing: 1, textTransform: 'uppercase', fontWeight: 500,
  };

  const fieldStyle: React.CSSProperties = {
    display: 'flex', alignItems: 'center', gap: 10, padding: '0 16px',
    height: 48, borderRadius: 12,
    background: 'rgba(255,255,255,0.03)',
    border: '1px solid rgba(255,255,255,0.08)',
    transition: 'border-color 0.2s, box-shadow 0.2s',
  };

  const inputStyle: React.CSSProperties = {
    flex: 1, height: '100%', background: 'none', border: 'none',
    outline: 'none', fontSize: 15, color: '#e2e8f0', fontFamily: 'inherit',
  };

  return (
    <div style={{
      minHeight: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center',
      background: 'linear-gradient(135deg, #0a0a1a 0%, #1a1040 30%, #0d1b2a 60%, #0a0a1a 100%)',
      position: 'relative', overflow: 'hidden', fontFamily: "'Noto Sans SC', 'PingFang SC', system-ui, sans-serif",
    }}>
      {/* Ambient orbs */}
      <div style={{ position: 'absolute', top: '-15%', right: '-10%', width: '60vw', height: '60vw', background: 'radial-gradient(circle, rgba(139,92,246,0.10) 0%, transparent 70%)', borderRadius: '50%', pointerEvents: 'none' }} />
      <div style={{ position: 'absolute', bottom: '-20%', left: '-5%', width: '50vw', height: '50vw', background: 'radial-gradient(circle, rgba(56,189,248,0.06) 0%, transparent 70%)', borderRadius: '50%', pointerEvents: 'none' }} />

      {/* Glass card */}
      <div style={{
        width: 420, padding: '48px 40px 40px', borderRadius: 24,
        background: 'rgba(255,255,255,0.04)',
        backdropFilter: 'blur(24px)', WebkitBackdropFilter: 'blur(24px)',
        border: '1px solid rgba(255,255,255,0.08)',
        boxShadow: '0 24px 80px rgba(0,0,0,0.4), 0 0 1px rgba(255,255,255,0.06) inset',
        position: 'relative', zIndex: 1, overflow: 'visible',
        animation: 'fadeIn 0.6s ease-out',
      }}>
        {/* Top accent */}
        <div style={{ position: 'absolute', top: 0, left: 60, right: 60, height: 1, background: 'linear-gradient(90deg, transparent, rgba(139,92,246,0.4), rgba(56,189,248,0.4), transparent)' }} />

        {/* Title */}
        <div style={{ textAlign: 'center', marginBottom: 32, animation: 'slideDown 0.5s ease-out 0.1s both' }}>
          <div style={{ display: 'inline-flex', alignItems: 'center', justifyContent: 'center', width: 56, height: 56, borderRadius: 16, marginBottom: 16, background: 'linear-gradient(135deg, rgba(139,92,246,0.2), rgba(56,189,248,0.15))', border: '1px solid rgba(139,92,246,0.2)' }}>
            <span style={{ fontSize: 26 }}>✨</span>
          </div>
          <h1 style={{ margin: 0, fontSize: 26, fontWeight: 700, background: 'linear-gradient(135deg, #e2e8f0 0%, #c4b5fd 50%, #7dd3fc 100%)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>创建账号</h1>
          <p style={{ margin: '8px 0 0', fontSize: 13, color: 'rgba(148,163,184,0.5)' }}>加入竞拍，发现好物</p>
        </div>

        <form onSubmit={handleSubmit}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            {/* Avatar Upload */}
            <div style={{ ...inputWrapperStyle(0.1), display: 'flex', justifyContent: 'center' }}>
              <label style={{ cursor: 'pointer', position: 'relative' }}>
                <input type="file" accept="image/*" onChange={handleAvatarChange} style={{ display: 'none' }} />
                <div style={{
                  width: 72, height: 72, borderRadius: '50%', overflow: 'hidden',
                  background: 'rgba(255,255,255,0.05)',
                  border: '2px dashed rgba(255,255,255,0.12)',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  transition: 'border-color 0.2s', position: 'relative',
                }}
                  onMouseEnter={e => (e.currentTarget.style.borderColor = 'rgba(139,92,246,0.5)')}
                  onMouseLeave={e => (e.currentTarget.style.borderColor = 'rgba(255,255,255,0.12)')}
                >
                  {avatarPreview ? (
                    <img src={avatarPreview} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                  ) : (
                    <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="rgba(148,163,184,0.4)" strokeWidth="1.5">
                      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/>
                    </svg>
                  )}
                </div>
                <div style={{ textAlign: 'center', fontSize: 11, color: 'rgba(148,163,184,0.5)', marginTop: 6 }}>点击上传头像</div>
              </label>
            </div>

            <div style={inputWrapperStyle(0.15)}>
              <label style={labelStyle}>用户名</label>
              <div style={fieldStyle} onFocusCapture={e => { e.currentTarget.style.borderColor='rgba(139,92,246,0.4)'; e.currentTarget.style.boxShadow='0 0 0 3px rgba(139,92,246,0.08)'; }} onBlurCapture={e => { e.currentTarget.style.borderColor='rgba(255,255,255,0.08)'; e.currentTarget.style.boxShadow='none'; }}>
                <input placeholder="请输入用户名" value={form.username} onChange={e => setForm({ ...form, username: e.target.value })} required style={inputStyle} />
              </div>
            </div>

            <div style={inputWrapperStyle(0.2)}>
              <label style={labelStyle}>昵称</label>
              <div style={fieldStyle} onFocusCapture={e => { e.currentTarget.style.borderColor='rgba(139,92,246,0.4)'; e.currentTarget.style.boxShadow='0 0 0 3px rgba(139,92,246,0.08)'; }} onBlurCapture={e => { e.currentTarget.style.borderColor='rgba(255,255,255,0.08)'; e.currentTarget.style.boxShadow='none'; }}>
                <input placeholder="请输入昵称" value={form.nickname} onChange={e => setForm({ ...form, nickname: e.target.value })} required style={inputStyle} />
              </div>
            </div>

            <div style={inputWrapperStyle(0.25)}>
              <label style={labelStyle}>密码</label>
              <div style={fieldStyle} onFocusCapture={e => { e.currentTarget.style.borderColor='rgba(139,92,246,0.4)'; e.currentTarget.style.boxShadow='0 0 0 3px rgba(139,92,246,0.08)'; }} onBlurCapture={e => { e.currentTarget.style.borderColor='rgba(255,255,255,0.08)'; e.currentTarget.style.boxShadow='none'; }}>
                <input type="password" placeholder="请输入密码" value={form.password} onChange={e => setForm({ ...form, password: e.target.value })} required style={inputStyle} />
              </div>
            </div>

            <div style={{ ...inputWrapperStyle(0.3), position: 'relative', zIndex: 100 }}>
              <label style={labelStyle}>角色</label>
              <div style={{ position: 'relative' }}>
                <div onClick={() => setShowRoleMenu(!showRoleMenu)} style={{
                  ...fieldStyle, cursor: 'pointer', justifyContent: 'space-between',
                }}>
                  <span style={{ fontSize: 15, color: '#e2e8f0' }}>{roleLabels[form.role]}</span>
                  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="rgba(148,163,184,0.4)" strokeWidth="2"
                    style={{ transform: showRoleMenu ? 'rotate(180deg)' : 'rotate(0)', transition: 'transform 0.2s' }}>
                    <path d="M6 9l6 6 6-6"/>
                  </svg>
                </div>
                <div style={{
                  position: 'absolute', top: 56, left: 0, right: 0, zIndex: 100,
                  background: 'rgba(15,15,40,0.98)', backdropFilter: 'blur(16px)',
                  borderRadius: 12, border: '1px solid rgba(255,255,255,0.08)',
                  overflow: 'hidden',
                  opacity: showRoleMenu ? 1 : 0,
                  transform: showRoleMenu ? 'translateY(0)' : 'translateY(-8px)',
                  pointerEvents: showRoleMenu ? 'auto' : 'none',
                  transition: 'opacity 0.25s ease, transform 0.25s ease',
                }}>
                  {[0, 1].map(r => (
                    <div key={r} onClick={() => { setForm({ ...form, role: r }); setShowRoleMenu(false); }} style={{
                      padding: '12px 16px', cursor: 'pointer', fontSize: 15, color: form.role === r ? '#a78bfa' : '#cbd5e0',
                      background: form.role === r ? 'rgba(139,92,246,0.1)' : 'transparent',
                      transition: 'background 0.15s',
                    }}
                      onMouseEnter={e => { if (form.role !== r) e.currentTarget.style.background = 'rgba(255,255,255,0.04)'; }}
                      onMouseLeave={e => { if (form.role !== r) e.currentTarget.style.background = 'transparent'; }}
                    >{roleLabels[r]}</div>
                  ))}
                </div>
              </div>
              {/* Close on click outside */}
              {showRoleMenu && <div onClick={() => setShowRoleMenu(false)} style={{ position: 'fixed', inset: 0, zIndex: 5 }} />}
            </div>

            {error && (
              <div style={{ color: '#fca5a5', fontSize: 13, textAlign: 'center', padding: '10px 16px', borderRadius: 8, background: 'rgba(239,68,68,0.1)', border: '1px solid rgba(239,68,68,0.15)', animation: 'shake 0.4s ease-out' }}>{error}</div>
            )}

            <button type="submit" disabled={loading}
              onMouseEnter={e => { if(!loading){ e.currentTarget.style.transform='translateY(-1px)'; e.currentTarget.style.boxShadow='0 6px 28px rgba(139,92,246,0.45)'; }}}
              onMouseLeave={e => { e.currentTarget.style.transform='translateY(0)'; e.currentTarget.style.boxShadow='0 4px 20px rgba(139,92,246,0.3)'; }}
              style={{
                marginTop: 4, height: 48, borderRadius: 12, border: 'none',
                cursor: loading ? 'not-allowed' : 'pointer', fontSize: 15, fontWeight: 600,
                color: '#fff', position: 'relative', overflow: 'hidden',
                background: loading ? 'linear-gradient(135deg, rgba(139,92,246,0.4), rgba(56,189,248,0.3))' : 'linear-gradient(135deg, #8b5cf6, #6366f1)',
                boxShadow: loading ? 'none' : '0 4px 20px rgba(139,92,246,0.3)',
                transition: 'all 0.25s ease',
                animation: 'slideUp 0.5s ease-out 0.35s both',
              }}>
              <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', height: '100%', background: 'linear-gradient(90deg, transparent, rgba(255,255,255,0.08), transparent)', transform: 'skewX(-20deg) translateX(-100%)', animation: 'btnShine 3s infinite' }} />
              {loading ? '注册中...' : '创建账号'}
            </button>

            <div style={{ textAlign: 'center', animation: 'slideUp 0.5s ease-out 0.4s both' }}>
              <span style={{ fontSize: 13, color: 'rgba(148,163,184,0.4)' }}>已有账号？</span>
              <Link to="/login" style={{ fontSize: 13, color: '#a78bfa', textDecoration: 'none', marginLeft: 4, fontWeight: 500 }}
                onMouseEnter={e => (e.currentTarget.style.color = '#c4b5fd')}
                onMouseLeave={e => (e.currentTarget.style.color = '#a78bfa')}
              >去登录</Link>
            </div>
          </div>
        </form>
      </div>

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
