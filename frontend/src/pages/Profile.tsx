import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../store/AuthContext';

export default function Profile() {
  const { user, token } = useAuth();
  const navigate = useNavigate();
  const [nickname, setNickname] = useState(user?.nickname || '');
  const [avatar, setAvatar] = useState<File | null>(null);
  const [avatarPreview, setAvatarPreview] = useState(user?.avatar || '');
  const [saving, setSaving] = useState(false);
  const [showSuccess, setShowSuccess] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  const handleAvatar = (e: React.ChangeEvent<HTMLInputElement>) => {
    const f = e.target.files?.[0];
    if (!f) return;
    setAvatar(f);
    const r = new FileReader();
    r.onload = () => setAvatarPreview(r.result as string);
    r.readAsDataURL(f);
  };

  const handleSave = async () => {
    if (!token || !user) return;
    setSaving(true);

    try {
      // Upload avatar if changed
      let avatarUrl = user.avatar || '';
      if (avatar) {
        const form = new FormData();
        form.append('file', avatar);
        const res = await fetch('/api/upload/image', {
          method: 'POST',
          headers: { Authorization: `Bearer ${token}` },
          body: form,
        });
        const json = await res.json();
        if (json.code !== 0) throw new Error(json.message);
        avatarUrl = json.data.url;
      }

      // Update user profile
      const updates: Record<string, any> = {};
      if (nickname.trim() && nickname !== user.nickname) updates.nickname = nickname.trim();
      if (avatarUrl !== user.avatar) updates.avatar = avatarUrl;

      if (Object.keys(updates).length > 0) {
        const res = await fetch(`/api/users/${user.id}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
          body: JSON.stringify(updates),
        });
        const json = await res.json();
        if (json.code !== 0) throw new Error(json.message);

        // Update local state
        const savedUser = { ...user, ...updates };
        localStorage.setItem('user', JSON.stringify(savedUser));
      }

      setShowSuccess(true);
      setTimeout(() => { setShowSuccess(false); window.location.reload(); }, 1800);
    } catch (err: any) {
      setErrorMsg(err.message || '保存失败');
    }
    setSaving(false);
  };

  return (
    <div style={{
      minHeight: '100vh',
      background: 'linear-gradient(180deg, #0a0a1a 0%, #0d1b2a 50%, #0f172a 100%)',
      fontFamily: "'Noto Sans SC', 'PingFang SC', system-ui, sans-serif",
      color: '#e2e8f0',
    }}>
      <div style={{ position: 'fixed', top: '-10%', right: '-10%', width: '40vw', height: '40vw', background: 'radial-gradient(circle, rgba(99,102,241,0.06) 0%, transparent 70%)', borderRadius: '50%', pointerEvents: 'none' }} />

      <div style={{ maxWidth: 420, margin: '0 auto', padding: '60px 20px', position: 'relative', zIndex: 1 }}>
        {/* Success modal */}
        {/* Success toast */}
        <div style={{
          position: 'fixed', top: showSuccess ? 24 : -80, left: '50%', transform: 'translateX(-50%)',
          zIndex: 500, pointerEvents: showSuccess ? 'auto' : 'none',
          background: 'rgba(15,15,40,0.95)', backdropFilter: 'blur(16px)',
          borderRadius: 14, padding: '14px 28px',
          border: '1px solid rgba(255,255,255,0.08)',
          boxShadow: '0 8px 32px rgba(0,0,0,0.4)',
          fontSize: 15, fontWeight: 600, color: '#e2e8f0',
          transition: 'top 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275)',
        }}>
          {showSuccess ? '保存成功' : ''}
        </div>

        {/* Error modal */}
        {errorMsg && (
          <div style={{
            position: 'fixed', inset: 0, zIndex: 500,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: 'rgba(0,0,0,0.5)', backdropFilter: 'blur(4px)',
          }} onClick={() => setErrorMsg('')}>
            <div style={{
              background: 'rgba(15,15,40,0.98)', backdropFilter: 'blur(20px)',
              borderRadius: 16, padding: '28px 36px',
              border: '1px solid rgba(255,255,255,0.08)',
              textAlign: 'center', maxWidth: 300,
              animation: 'fadeIn 0.3s ease-out',
            }} onClick={e => e.stopPropagation()}>
              <div style={{ fontSize: 16, fontWeight: 700, color: '#fca5a5', marginBottom: 8 }}>保存失败</div>
              <div style={{ fontSize: 13, color: 'rgba(148,163,184,0.5)' }}>{errorMsg}</div>
              <button onClick={() => setErrorMsg('')} style={{
                marginTop: 16, padding: '8px 32px',
                background: 'rgba(255,255,255,0.06)', color: 'rgba(226,232,240,0.6)',
                border: '1px solid rgba(255,255,255,0.08)', borderRadius: 10,
                fontSize: 13, cursor: 'pointer',
              }}>关闭</button>
            </div>
          </div>
        )}

        <style>{`@keyframes fadeIn { from { opacity: 0 } to { opacity: 1 } }`}</style>

        {/* Header */}
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 36, gap: 16 }}>
          <button onClick={() => navigate(-1)} style={{
            background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)',
            borderRadius: 12, width: 40, height: 40, fontSize: 18, cursor: 'pointer',
            color: 'rgba(148,163,184,0.7)', display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}>←</button>
          <h1 style={{
            margin: 0, fontSize: 24, fontWeight: 700,
            background: 'linear-gradient(135deg, #e2e8f0, #a5b4fc)',
            WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent',
          }}>个人中心</h1>
        </div>

        {/* Avatar */}
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <label style={{ cursor: 'pointer', display: 'inline-block' }}>
            <input type="file" accept="image/*" onChange={handleAvatar} style={{ display: 'none' }} />
            <div style={{
              width: 96, height: 96, borderRadius: '50%', margin: '0 auto',
              border: '3px solid rgba(255,255,255,0.12)',
              overflow: 'hidden', position: 'relative',
              background: 'linear-gradient(135deg, #667eea, #764ba2)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontSize: 36, color: '#fff',
              transition: 'border-color 0.2s',
            }}
              onMouseEnter={e => { e.currentTarget.style.borderColor = 'rgba(99,102,241,0.5)'; }}
              onMouseLeave={e => { e.currentTarget.style.borderColor = 'rgba(255,255,255,0.12)'; }}
            >
              {avatarPreview ? (
                <img src={avatarPreview} alt="" style={{
                  position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover',
                }} />
              ) : (
                <span>{user?.nickname?.[0] || 'U'}</span>
              )}
              <div style={{
                position: 'absolute', bottom: 0, right: 0,
                width: 28, height: 28, borderRadius: '50%',
                background: '#6366f1', color: '#fff',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                fontSize: 14, border: '2px solid #1a1a2e',
              }}>📷</div>
            </div>
          </label>
          <div style={{ fontSize: 12, color: 'rgba(148,163,184,0.4)', marginTop: 8 }}>点击更换头像</div>
        </div>

        {/* Nickname */}
        <div style={{ marginBottom: 24 }}>
          <label style={{ fontSize: 12, color: 'rgba(148,163,184,0.6)', display: 'block', marginBottom: 6, letterSpacing: 0.5 }}>昵称</label>
          <input
            value={nickname}
            onChange={e => setNickname(e.target.value)}
            maxLength={20}
            style={{
              display: 'block', width: '100%', padding: '12px 16px',
              background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)',
              borderRadius: 12, fontSize: 15, color: '#e2e8f0', outline: 'none',
              fontFamily: 'inherit', boxSizing: 'border-box',
            }}
          />
        </div>

        {/* Username (read-only) */}
        <div style={{ marginBottom: 24 }}>
          <label style={{ fontSize: 12, color: 'rgba(148,163,184,0.6)', display: 'block', marginBottom: 6, letterSpacing: 0.5 }}>用户名</label>
          <div style={{
            padding: '12px 16px',
            background: 'rgba(255,255,255,0.02)', border: '1px solid rgba(255,255,255,0.04)',
            borderRadius: 12, fontSize: 15, color: 'rgba(148,163,184,0.5)',
          }}>{user?.username || '-'}</div>
        </div>

        {/* Save */}
        <button onClick={handleSave} disabled={saving} style={{
          width: '100%', padding: '14px', border: 'none', borderRadius: 12,
          fontSize: 15, fontWeight: 600, cursor: saving ? 'default' : 'pointer',
          color: '#fff', opacity: saving ? 0.6 : 1,
          background: saving
            ? 'linear-gradient(135deg, rgba(99,102,241,0.4), rgba(59,130,246,0.3))'
            : 'linear-gradient(135deg, #6366f1, #3b82f6)',
          transition: 'all 0.2s',
        }}
          onMouseEnter={e => { if(!saving) { e.currentTarget.style.transform='translateY(-1px)'; e.currentTarget.style.boxShadow='0 4px 20px rgba(99,102,241,0.3)'; }}}
          onMouseLeave={e => { e.currentTarget.style.transform='translateY(0)'; e.currentTarget.style.boxShadow='none'; }}
        >{saving ? '保存中...' : '保存'}</button>
      </div>
    </div>
  );
}
