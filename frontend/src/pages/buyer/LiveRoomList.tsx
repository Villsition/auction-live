import { useState, useEffect, useRef, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../../store/AuthContext';
import { publicApi } from '../../api';
import type { LiveRoom } from '../../types';

export default function LiveRoomList() {
  const [rooms, setRooms] = useState<LiveRoom[]>([]);
  const [loading, setLoading] = useState(true);
  const [showMenu, setShowMenu] = useState(false);
  const [keyword, setKeyword] = useState('');
  const [mousePos, setMousePos] = useState({ x: 0, y: 0 });
  const menuRef = useRef<HTMLDivElement>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout>>();
  const headerRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();
  const { user, logout } = useAuth();

  const fetchRooms = useCallback((kw: string) => {
    setLoading(true);
    publicApi.liveRooms(1, kw || undefined)
      .then((data: any) => {
        const list: LiveRoom[] = data.list || data || [];
        setRooms(list.filter(r => r.status === 1));
      })
      .catch(console.error)
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => { fetchRooms(''); }, [fetchRooms]);

  const handleSearch = (value: string) => {
    setKeyword(value);
    clearTimeout(timerRef.current);
    timerRef.current = setTimeout(() => fetchRooms(value), 300);
  };

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowMenu(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  return (
    <div style={{
      minHeight: '100vh',
      background: 'linear-gradient(180deg, #0a0a1a 0%, #111030 5%, #0d1b2a 30%, #0f172a 100%)',
      fontFamily: "'Noto Sans SC', 'PingFang SC', system-ui, sans-serif",
      position: 'relative',
    }}>
      {/* ===== Dynamic Header Area ===== */}
      <div
        ref={headerRef}
        onMouseMove={e => {
          if (headerRef.current) {
            const rect = headerRef.current.getBoundingClientRect();
            setMousePos({ x: e.clientX - rect.left, y: e.clientY - rect.top });
          }
        }}
        style={{
          position: 'relative', padding: '60px 24px 80px',
          background: 'linear-gradient(180deg, #0f0f2e 0%, #0d1b3a 60%, transparent 100%)',
          overflow: 'hidden',
        }}
      >
        {/* Animated glow follows cursor — smooth via CSS transform on parent */}
        <div style={{
          position: 'absolute', width: 500, height: 500,
          borderRadius: '50%',
          background: 'radial-gradient(circle, rgba(99,102,241,0.12) 0%, transparent 70%)',
          transform: `translate(${mousePos.x - 250}px, ${mousePos.y - 250}px)`,
          willChange: 'transform',
          pointerEvents: 'none',
        }} />
        <div style={{
          position: 'absolute', width: 300, height: 300,
          borderRadius: '50%',
          background: 'radial-gradient(circle, rgba(56,189,248,0.08) 0%, transparent 70%)',
          transform: `translate(${mousePos.x - 350}px, ${mousePos.y - 120}px)`,
          willChange: 'transform',
          pointerEvents: 'none',
        }} />

        {/* Header content */}
        <div style={{ maxWidth: 1400, margin: '0 auto', position: 'relative', zIndex: 1 }}>
          {/* Top row: avatar right */}
          <div style={{
            display: 'flex', justifyContent: 'flex-end', marginBottom: 40,
          }}>
            <div ref={menuRef} style={{ position: 'relative', zIndex: 1000 }}>
              {user?.avatar && user.avatar !== '' ? (
                <img src={user.avatar} alt=""
                  onMouseEnter={() => { clearTimeout(timerRef.current); setShowMenu(true); }}
                  onMouseLeave={() => { timerRef.current = setTimeout(() => setShowMenu(false), 150); }}
                  onClick={() => setShowMenu(!showMenu)}
                  style={{
                    width: 42, height: 42, borderRadius: '50%', cursor: 'pointer',
                    border: '2px solid rgba(255,255,255,0.15)', objectFit: 'cover',
                  }}
                />
              ) : (
                <div
                  onMouseEnter={() => { clearTimeout(timerRef.current); setShowMenu(true); }}
                  onMouseLeave={() => { timerRef.current = setTimeout(() => setShowMenu(false), 150); }}
                  onClick={() => setShowMenu(!showMenu)}
                  style={{
                    width: 42, height: 42, borderRadius: '50%', cursor: 'pointer',
                    border: '2px solid rgba(255,255,255,0.15)',
                    background: 'linear-gradient(135deg, #667eea, #764ba2)',
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    color: '#fff', fontSize: 18, fontWeight: 700,
                  }}
                >{user?.nickname?.[0] || 'U'}</div>
              )}
              <div
                onMouseEnter={() => clearTimeout(timerRef.current)}
                onMouseLeave={() => setShowMenu(false)}
                style={{
                  position: 'absolute', top: 52, right: 0,
                  background: 'rgba(15,15,40,0.95)', borderRadius: 12, backdropFilter: 'blur(20px)',
                  border: '1px solid rgba(255,255,255,0.08)',
                  boxShadow: '0 12px 40px rgba(0,0,0,0.5)',
                  overflow: 'hidden', minWidth: 150,
                  opacity: showMenu ? 1 : 0,
                  transform: showMenu ? 'translateY(0)' : 'translateY(-8px)',
                  transition: 'opacity 0.2s ease, transform 0.2s ease',
                  pointerEvents: showMenu ? 'auto' : 'none',
                }}
              >
                <div style={{ padding: '14px 18px', borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
                  <div style={{ fontWeight: 'bold', fontSize: 14, color: '#e2e8f0' }}>{user?.nickname}</div>
                  <div style={{ fontSize: 12, color: 'rgba(148,163,184,0.5)', marginTop: 2 }}>@{user?.username}</div>
                </div>
                <div onClick={() => { setShowMenu(false); navigate('/bid-history'); }} style={{
                  padding: '11px 18px', cursor: 'pointer', fontSize: 13, color: '#cbd5e0',
                  display: 'flex', alignItems: 'center', gap: 8, transition: 'background 0.15s',
                }}
                  onMouseEnter={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.04)')}
                  onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                >📜 历史竞拍</div>
                <div onClick={() => { setShowMenu(false); navigate('/profile'); }} style={{
                  padding: '11px 18px', cursor: 'pointer', fontSize: 13, color: '#cbd5e0',
                  display: 'flex', alignItems: 'center', gap: 8, transition: 'background 0.15s',
                }}
                  onMouseEnter={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.04)')}
                  onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                >👤 个人中心</div>
                <div onClick={() => { setShowMenu(false); navigate('/orders'); }} style={{
                  padding: '11px 18px', cursor: 'pointer', fontSize: 13, color: '#cbd5e0',
                  display: 'flex', alignItems: 'center', gap: 8, transition: 'background 0.15s',
                }}
                  onMouseEnter={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.04)')}
                  onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                >📋 我的订单</div>
                <div onClick={() => { setShowMenu(false); logout(); }} style={{
                  padding: '11px 18px', cursor: 'pointer', fontSize: 13, color: '#fca5a5',
                  display: 'flex', alignItems: 'center', gap: 8, transition: 'background 0.15s',
                }}
                  onMouseEnter={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.04)')}
                  onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                >🚪 退出登录</div>
              </div>
            </div>
          </div>

          {/* Title */}
          <div style={{ textAlign: 'center', marginBottom: 32 }}>
            <h1 style={{
              margin: 0, fontSize: 36, fontWeight: 800, letterSpacing: 2,
              background: 'linear-gradient(135deg, #e2e8f0 0%, #a5b4fc 40%, #7dd3fc 100%)',
              WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent',
            }}>竞拍直播间</h1>
            <p style={{ margin: '10px 0 0', fontSize: 14, color: 'rgba(148,163,184,0.5)', letterSpacing: 1 }}>
              发现好物 · 实时竞拍
            </p>
          </div>

          {/* Search */}
          <div style={{ display: 'flex', justifyContent: 'center' }}>
            <div style={{
              position: 'relative', width: 420,
            }}>
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="rgba(148,163,184,0.4)" strokeWidth="2"
                style={{ position: 'absolute', left: 16, top: '50%', transform: 'translateY(-50%)', pointerEvents: 'none' }}>
                <circle cx="11" cy="11" r="8"/><path d="M21 21l-4.35-4.35"/>
              </svg>
              <input
                type="text"
                value={keyword}
                onChange={e => handleSearch(e.target.value)}
                placeholder="搜索直播间或主播名称..."
                style={{
                  width: '100%', padding: '14px 20px 14px 46px',
                  background: 'rgba(255,255,255,0.04)',
                  backdropFilter: 'blur(10px)',
                  border: '1px solid rgba(255,255,255,0.08)',
                  borderRadius: 16, fontSize: 14, color: '#e2e8f0',
                  outline: 'none', fontFamily: 'inherit',
                  transition: 'border-color 0.2s, box-shadow 0.2s',
                }}
                onFocus={e => {
                  e.target.style.borderColor = 'rgba(99,102,241,0.5)';
                  e.target.style.boxShadow = '0 0 0 4px rgba(99,102,241,0.08)';
                }}
                onBlur={e => {
                  e.target.style.borderColor = 'rgba(255,255,255,0.08)';
                  e.target.style.boxShadow = 'none';
                }}
              />
            </div>
          </div>
        </div>
      </div>

      {/* ===== Room Grid ===== */}
      <div style={{ padding: '0 24px 60px', position: 'relative', zIndex: 1, marginTop: -20 }}>
        {loading ? (
          <div style={{ textAlign: 'center', padding: 60, color: 'rgba(148,163,184,0.4)', fontSize: 14 }}>
            加载中...
          </div>
        ) : rooms.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 80, color: 'rgba(148,163,184,0.4)' }}>
            <div style={{ fontSize: 56, marginBottom: 16 }}>📭</div>
            <div style={{ fontSize: 15, color: 'rgba(148,163,184,0.5)' }}>
              {keyword ? '未找到匹配的直播间' : '暂无直播中的直播间'}
            </div>
          </div>
        ) : (
          <div style={{
            maxWidth: 1400, margin: '0 auto',
            display: 'grid',
            gridTemplateColumns: 'repeat(5, 1fr)',
            gap: 16,
          }}>
            {rooms.map((room, i) => (
              <div key={room.id}
                onClick={() => navigate(`/room/${room.id}`)}
                style={{
                  background: 'rgba(255,255,255,0.03)',
                  backdropFilter: 'blur(10px)',
                  borderRadius: 14, overflow: 'hidden',
                  cursor: 'pointer',
                  border: '1px solid rgba(255,255,255,0.06)',
                  transition: 'transform 0.3s ease, box-shadow 0.3s ease',
                  animation: `cardIn 0.4s ease-out ${i * 0.04}s both`,
                }}
                onMouseEnter={e => {
                  e.currentTarget.style.transform = 'translateY(-6px) scale(1.03)';
                  e.currentTarget.style.boxShadow = '0 20px 48px rgba(0,0,0,0.5), 0 0 0 2px rgba(99,102,241,0.3), 0 0 20px rgba(99,102,241,0.1)';
                  e.currentTarget.style.borderColor = 'rgba(99,102,241,0.4)';
                }}
                onMouseLeave={e => {
                  e.currentTarget.style.transform = 'translateY(0) scale(1)';
                  e.currentTarget.style.boxShadow = 'none';
                  e.currentTarget.style.borderColor = 'rgba(255,255,255,0.06)';
                }}
              >
                <div style={{
                  aspectRatio: '16 / 9',
                  background: room.cover_image ? `url(${room.cover_image})` : '#1a1a3e',
                  backgroundSize: 'cover', backgroundPosition: 'center',
                  display: 'flex', alignItems: 'flex-end', justifyContent: 'flex-end',
                  padding: 10, position: 'relative',
                }}>
                  {!room.cover_image && (
                    <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'linear-gradient(135deg, #1a1a3e, #0f0f2e)' }}>
                      <span style={{ fontSize: 40, opacity: 0.3 }}>📺</span>
                    </div>
                  )}
                  <span style={{ position: 'relative', zIndex: 1, background: 'rgba(0,0,0,0.5)', color: '#fff', padding: '3px 8px', borderRadius: 6, fontSize: 11, fontWeight: 500 }}>
                    👥 {room.online_count ?? 0}
                  </span>
                </div>
                <div style={{ padding: '12px 14px' }}>
                  <div style={{
                    fontSize: 14, fontWeight: 600, color: '#e2e8f0',
                    marginBottom: 8, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                  }}>
                    {room.title}
                  </div>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <img
                      src={room.seller_avatar || undefined}
                      alt=""
                      style={{ width: 26, height: 26, borderRadius: '50%', background: 'rgba(99,102,241,0.3)', border: '1px solid rgba(255,255,255,0.08)' }}
                      onError={(e) => { (e.target as HTMLImageElement).src = 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><circle cx="16" cy="16" r="16" fill="%2333386b"/></svg>'; }}
                    />
                    <span style={{ fontSize: 12, color: 'rgba(148,163,184,0.6)' }}>
                      {room.seller_nickname || `主播${room.seller_id}`}
                    </span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <style>{`
        @keyframes cardIn { from { opacity: 0; transform: translateY(20px); } to { opacity: 1; transform: translateY(0); } }
      `}</style>
    </div>
  );
}
