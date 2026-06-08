import { BrowserRouter, Routes, Route, Navigate, Link, useLocation, useParams } from 'react-router-dom';
import { useState, useEffect, useRef } from 'react';
import { AuthProvider, useAuth } from './store/AuthContext';
import Login from './pages/Login';
import Register from './pages/Register';
import LiveRoomList from './pages/buyer/LiveRoomList';
import BuyerLiveRoom from './pages/buyer/LiveRoom';
import BuyerOrders from './pages/buyer/Orders';
import BidHistory from './pages/buyer/BidHistory';
import Dashboard from './pages/seller/Dashboard';
import Products from './pages/seller/Products';
import SellerLiveRooms from './pages/seller/LiveRooms';
import SellerOrders from './pages/seller/Orders';
import Profile from './pages/Profile';
import { seller as sellerApi } from './api';
import type { LiveRoom } from './types';

function Home() {
  const { isSeller } = useAuth();
  return isSeller ? <SellerLayout><Dashboard /></SellerLayout> : <LiveRoomList />;
}

function RoomWrapper() {
  const { roomId } = useParams();
  return <BuyerLiveRoom key={roomId} />;
}

function Protected({ children }: { children: React.ReactNode }) {
  const { token } = useAuth();
  if (!token) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

function SellerLayout({ children }: { children: React.ReactNode }) {
  const location = useLocation();
  const { user, token, logout } = useAuth();
  const [liveRoomId, setLiveRoomId] = useState<number | null>(null);
  const [showMenu, setShowMenu] = useState(false);
  const menuTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Fetch seller's live room — only show "直播间" tab when live
  useEffect(() => {
    if (!token || (user?.role ?? 0) < 1) return;
    sellerApi.listRooms(token).then((data: any) => {
      const list: LiveRoom[] = data?.list || data || [];
      const live = list.find((r: LiveRoom) => r.status === 1);
      setLiveRoomId(live ? live.id : null);
    }).catch(() => {});
  }, [token, user?.role]);

  const tabs = [
    { path: '/', label: '工作台' },
    { path: '/seller/products', label: '商品管理' },
    ...(liveRoomId ? [{ path: `/room/${liveRoomId}`, label: '直播间' }] : []),
  ];

  return (
    <div style={{ minHeight: '100vh', background: 'linear-gradient(180deg,#0a0a1a 0%,#0d1b2a 50%,#0f172a 100%)', fontFamily:"'Noto Sans SC','PingFang SC',system-ui,sans-serif" }}>
      {/* Top Nav — dark glass */}
      <div style={{ background:'rgba(10,10,26,0.8)', backdropFilter:'blur(20px)', borderBottom:'1px solid rgba(255,255,255,0.06)', padding:'0 24px', position:'sticky', top:0, zIndex:50 }}>
        <div style={{ maxWidth:1200, margin:'0 auto', display:'flex', alignItems:'center', justifyContent:'space-between' }}>
          <div style={{ display:'flex', gap:0 }}>
            {tabs.map(t => {
              const isActive = t.path === '/'
                ? location.pathname === '/'
                : t.path === '/seller/products'
                  ? location.pathname.startsWith('/seller/products')
                  : location.pathname.startsWith('/room/');
              // Wrap Link+glow in a relative container for hover effects
              return (<div key={t.label} style={{ position:'relative' }}
                onMouseEnter={e=>{
                  const a=e.currentTarget.querySelector('a'); if(a){a.style.color='#e2e8f0';a.style.transform='scale(1.06)';}
                  const s=e.currentTarget.querySelector('.glow') as HTMLElement; if(s){s.style.opacity='1';s.style.boxShadow='0 -8px 20px rgba(99,102,241,0.4)';}
                }}
                onMouseLeave={e=>{
                  const a=e.currentTarget.querySelector('a'); if(a){a.style.color=isActive?'#e2e8f0':'rgba(148,163,184,0.6)';a.style.transform='scale(1)';}
                  const s=e.currentTarget.querySelector('.glow') as HTMLElement; if(s){s.style.opacity=isActive?'1':'0';s.style.boxShadow=isActive?'0 -6px 16px rgba(99,102,241,0.25)':'none';}
                }}
              >
                <Link to={t.path} style={{
                  padding:'16px 20px', fontSize:15, textDecoration:'none', display:'block',
                  color: isActive ? '#e2e8f0' : 'rgba(148,163,184,0.6)',
                  fontWeight: isActive ? 700 : 500,
                  transition:'all 0.25s ease',
                }}>{t.label}</Link>
                <span className="glow" style={{
                  position:'absolute', bottom:0, left:'15%', right:'15%', height:3,
                  borderRadius:'2px 2px 0 0',
                  background:'linear-gradient(0deg, #6366f1, #818cf8, transparent)',
                  opacity: isActive ? 1 : 0,
                  boxShadow: isActive ? '0 -6px 16px rgba(99,102,241,0.25)' : 'none',
                  transition:'all 0.3s ease',
                }} />
              </div>
            )})}
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, position: 'relative', marginRight: 16 }}>
            {user?.avatar && user.avatar !== '' ? (
              <img src={user.avatar} alt=""
                onMouseEnter={() => { if (menuTimerRef.current) clearTimeout(menuTimerRef.current); setShowMenu(true); }}
                onMouseLeave={() => { menuTimerRef.current = setTimeout(() => setShowMenu(false), 150); }}
                style={{ width: 36, height: 36, borderRadius: '50%', cursor: 'pointer', objectFit: 'cover', border: '2px solid #e2e8f0' }}
              />
            ) : (
              <div
                onMouseEnter={() => { if (menuTimerRef.current) clearTimeout(menuTimerRef.current); setShowMenu(true); }}
                onMouseLeave={() => { menuTimerRef.current = setTimeout(() => setShowMenu(false), 150); }}
                style={{ width: 36, height: 36, borderRadius: '50%', cursor: 'pointer', background: 'linear-gradient(135deg, #667eea, #764ba2)', display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#fff', fontSize: 16, fontWeight: 700, border: '2px solid #e2e8f0' }}
              >{user?.nickname?.[0] || 'U'}</div>
            )}
            <div
              onMouseEnter={() => { if (menuTimerRef.current) clearTimeout(menuTimerRef.current); }}
              onMouseLeave={() => setShowMenu(false)}
              style={{
                position: 'absolute', top: 46, right: 0, zIndex: 100,
                background: 'rgba(15,15,40,0.95)', borderRadius: 12, backdropFilter: 'blur(20px)',
                border: '1px solid rgba(255,255,255,0.08)',
                boxShadow: '0 12px 40px rgba(0,0,0,0.5)',
                overflow: 'hidden', minWidth: 150,
                opacity: showMenu ? 1 : 0,
                transform: showMenu ? 'translateY(0)' : 'translateY(-8px)',
                pointerEvents: showMenu ? 'auto' : 'none',
                transition: 'opacity 0.2s ease, transform 0.2s ease',
              }}
            >
              <div style={{ padding: '14px 18px', borderBottom: '1px solid rgba(255,255,255,0.06)' }}>
                <div style={{ fontWeight: 'bold', fontSize: 14, color: '#e2e8f0' }}>{user?.nickname}</div>
                <div style={{ fontSize: 12, color: 'rgba(148,163,184,0.5)', marginTop: 2 }}>@{user?.username}</div>
              </div>
              <div onClick={() => { window.location.href = '/profile'; }} style={{ padding: '11px 18px', cursor: 'pointer', fontSize: 13, color: '#cbd5e0', display: 'flex', alignItems: 'center', gap: 8 }}
                onMouseEnter={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.04)')}
                onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
              >👤 个人中心</div>
              <div onClick={logout} style={{ padding: '11px 18px', cursor: 'pointer', fontSize: 13, color: '#fca5a5', display: 'flex', alignItems: 'center', gap: 8 }}
                onMouseEnter={e => (e.currentTarget.style.background = 'rgba(255,255,255,0.04)')}
                onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
              >🚪 退出登录</div>
            </div>
          </div>
        </div>
      </div>
      {/* Content */}
      {children}
    </div>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/" element={<Protected><Home /></Protected>} />
          <Route path="/room/:roomId" element={<Protected><RoomWrapper /></Protected>} />
          <Route path="/orders" element={<Protected><BuyerOrders /></Protected>} />
          <Route path="/bid-history" element={<Protected><BidHistory /></Protected>} />
          <Route path="/seller/products" element={<Protected><SellerLayout><Products /></SellerLayout></Protected>} />
          <Route path="/seller/live-rooms" element={<Protected><SellerLayout><SellerLiveRooms /></SellerLayout></Protected>} />
          <Route path="/seller/orders" element={<Protected><SellerLayout><SellerOrders /></SellerLayout></Protected>} />
          <Route path="/profile" element={<Protected><Profile /></Protected>} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  );
}
