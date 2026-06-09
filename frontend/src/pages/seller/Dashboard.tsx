import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../../store/AuthContext';
import { seller } from '../../api';
import type { DashboardData, LiveRoom } from '../../types';

const fmt = (s?: string) => (s ? (s.endsWith('.00') ? s.slice(0, -3) : s.includes('.') ? s.replace(/0+$/, '').replace(/\.$/, '') : s) : '0');

export default function Dashboard() {
  const { user, token } = useAuth();
  const navigate = useNavigate();
  const [data, setData] = useState<DashboardData | null>(null);
  const [rooms, setRooms] = useState<LiveRoom[]>([]);
  const [loading, setLoading] = useState(true);
  const [toast, setToast] = useState('');

  // Create flow
  const [showCreate, setShowCreate] = useState(false);
  const [title, setTitle] = useState('');
  const [startRoom, setStartRoom] = useState<LiveRoom | null>(null);
  const [startTitle, setStartTitle] = useState('');
  const [bgVideoUrl, setBgVideoUrl] = useState('');
  const [uploadingVideo, setUploadingVideo] = useState(false);
  const [selectProducts, setSelectProducts] = useState<any[]>([]);
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set());
  const [loadingProducts, setLoadingProducts] = useState(false);
  const [hoveredCard, setHoveredCard] = useState<string | null>(null);

  const load = async () => {
    if (!token) return;
    Promise.all([seller.dashboard(token), seller.listRooms(token)])
      .then(([d, r]) => { setData(d); setRooms((r as any)?.list || r || []); })
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, [token]);

  useEffect(() => { if (!toast) return; const t = setTimeout(() => setToast(''), 2500); return () => clearTimeout(t); }, [toast]);

  const handleCreate = async () => {
    if (!token || !title.trim()) return;
    try {
      const room = await seller.createRoom({ title: title.trim() }, token);
      setShowCreate(false); setTitle('');
      setToast('✅ 直播间创建成功！');
      const r = await seller.listRooms(token);
      setRooms((r as any)?.list || r || []);
      setStartRoom(room); setStartTitle(room.title || title.trim());
    } catch (err: any) { alert(err.message); }
  };

  const handleStart = async () => {
    if (!token || !startRoom || !startTitle.trim()) return;
    if (selectedIds.size === 0) { alert('请至少选择一件商品'); return; }
    try {
      await fetch(`/api/seller/live-rooms/${startRoom.id}`, { method:'PUT', headers:{'Content-Type':'application/json', Authorization:`Bearer ${token}`}, body:JSON.stringify({title:startTitle.trim(), bg_video: bgVideoUrl}) });
      await seller.startLive(startRoom.id, token);
      const sorted = selectProducts.filter((p:any) => selectedIds.has(p.id));
      for (let i=0; i<sorted.length; i++) {
        const p = sorted[i];
        await seller.createAuction({ room_id:startRoom.id, product_id:p.id, start_price:p.start_price||'0', bid_increment:p.bid_increment||'10', ceiling_price:p.ceiling_price||'0', delay_seconds:p.delay_seconds||30, duration_min:p.duration_min||5, sort_order:i }, token);
      }
      setStartRoom(null); setSelectedIds(new Set()); setSelectProducts([]);
      setToast('🔴 直播已开启！'); load();
      navigate(`/room/${startRoom.id}`);
    } catch (err: any) { alert(err.message); }
  };

  const handleVideoUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]; if (!file || !token) return;
    setUploadingVideo(true);
    try { const url = await seller.uploadVideo(file, token); setBgVideoUrl(url); }
    catch (err: any) { alert('视频上传失败: ' + err.message); }
    setUploadingVideo(false);
  };

  const openStart = async (room: LiveRoom) => {
    setStartRoom(room); setStartTitle(room.title||''); setSelectedIds(new Set());
    setBgVideoUrl(room.bg_video || '');
    setLoadingProducts(true);
    try { const data = await seller.listProducts('page=1&page_size=50', token!); setSelectProducts((data.list||[]).filter((p:any)=>p.status===1)); }
    catch { setSelectProducts([]); }
    setLoadingProducts(false);
  };

  if (loading) return <div style={{ minHeight:'100vh', background:'linear-gradient(180deg,#0a0a1a 0%,#0d1b2a 50%,#0f172a 100%)', display:'flex', alignItems:'center', justifyContent:'center', color:'rgba(148,163,184,0.4)', fontFamily:"'Noto Sans SC','PingFang SC',system-ui,sans-serif" }}>加载中...</div>;
  if (!data) return <div style={{ minHeight:'100vh', background:'linear-gradient(180deg,#0a0a1a 0%,#0d1b2a 50%,#0f172a 100%)', display:'flex', alignItems:'center', justifyContent:'center', color:'#fca5a5', fontFamily:"'Noto Sans SC','PingFang SC',system-ui,sans-serif" }}>加载失败</div>;

  const hasRooms = rooms.length > 0;
  const liveRoom = rooms.find(r => r.status === 1);
  const offlineRoom = rooms.find(r => r.status !== 1);

  const stats = [
    { label: '总收入', value: `¥${fmt(data.revenue_total)}`, color: '#10b981', icon: '💰', link: '' },
    { label: '待支付', value: data.order_stats['unpaid']||0, color: '#f59e0b', icon: '⏳', link: '/seller/orders?status=0' },
    { label: '已售', value: data.auction_stats['sold']||0, color: '#8b5cf6', icon: '📦', link: '/seller/orders?status=1,2' },
    { label: '已完成', value: data.order_stats['completed']||0, color: '#10b981', icon: '✅', link: '/seller/orders?status=3' },
  ];

  return (
    <div style={{ minHeight:'100vh', background:'linear-gradient(180deg,#0a0a1a 0%,#0d1b2a 50%,#0f172a 100%)', fontFamily:"'Noto Sans SC','PingFang SC',system-ui,sans-serif", color:'#e2e8f0' }}>
      <div style={{ position:'fixed', top:'-10%', right:'-10%', width:'40vw', height:'40vw', background:'radial-gradient(circle,rgba(99,102,241,0.06) 0%,transparent 70%)', borderRadius:'50%', pointerEvents:'none' }} />

      <div style={{ maxWidth:900, margin:'0 auto', padding:'60px 20px 40px', position:'relative', zIndex:1 }}>
        {/* Toast */}
        {toast && <div style={{ position:'fixed', top:20, left:'50%', transform:'translateX(-50%)', zIndex:200, background:'rgba(15,15,40,0.95)', backdropFilter:'blur(12px)', color:'#fff', padding:'10px 24px', borderRadius:24, fontSize:14, border:'1px solid rgba(255,255,255,0.1)' }}>{toast}</div>}
        {/* Room card */}
        {/* Room card */}
        {hasRooms ? (
          <div style={{ background:'rgba(255,255,255,0.03)', backdropFilter:'blur(12px)', borderRadius:20, padding:'24px 28px', border:'1px solid rgba(255,255,255,0.06)', display:'flex', alignItems:'center', justifyContent:'space-between', marginBottom:28 }}>
            <div style={{ display:'flex', alignItems:'center', gap:16 }}>
              <div style={{ width:48, height:48, borderRadius:14, background:liveRoom?'rgba(239,68,68,0.15)':'rgba(148,163,184,0.1)', display:'flex', alignItems:'center', justifyContent:'center', fontSize:22 }}>{liveRoom?'🔴':'⭕'}</div>
              <div>
                <div style={{ fontWeight:700, fontSize:16 }}>{liveRoom?liveRoom.title:(offlineRoom?.title||rooms[0].title)}</div>
                <div style={{ fontSize:13, color:'rgba(148,163,184,0.5)', marginTop:3 }}>{liveRoom?'直播中':'离线'}</div>
              </div>
            </div>
            {liveRoom
              ? <button onClick={()=>navigate(`/room/${liveRoom.id}`)} style={{ padding:'10px 24px', background:'rgba(239,68,68,0.15)', color:'#fca5a5', border:'1px solid rgba(239,68,68,0.2)', borderRadius:12, cursor:'pointer', fontSize:14, fontWeight:600, transition:'all 0.2s' }} onMouseEnter={e=>{e.currentTarget.style.background='rgba(239,68,68,0.25)'}} onMouseLeave={e=>{e.currentTarget.style.background='rgba(239,68,68,0.15)'}}>进入直播间</button>
              : <button onClick={()=>openStart(offlineRoom||rooms[0])} style={{ padding:'10px 24px', background:'linear-gradient(135deg,#10b981,#059669)', color:'#fff', border:'none', borderRadius:12, cursor:'pointer', fontSize:14, fontWeight:600, transition:'all 0.2s' }} onMouseEnter={e=>{e.currentTarget.style.transform='scale(1.04)';e.currentTarget.style.boxShadow='0 4px 16px rgba(16,185,129,0.3)'}} onMouseLeave={e=>{e.currentTarget.style.transform='scale(1)';e.currentTarget.style.boxShadow='none'}}>开启直播</button>
            }
          </div>
        ) : (
          <div style={{ background:'rgba(255,255,255,0.02)', backdropFilter:'blur(12px)', borderRadius:20, padding:'48px 20px', textAlign:'center', border:'1px dashed rgba(255,255,255,0.08)', marginBottom:28 }}>
            <div style={{ fontSize:48, marginBottom:12, opacity:0.3 }}>📺</div>
            <div style={{ fontSize:17, fontWeight:700, color:'rgba(226,232,240,0.7)', marginBottom:6 }}>还没有直播间</div>
            <div style={{ fontSize:13, color:'rgba(148,163,184,0.4)', marginBottom:24 }}>创建直播间后即可开启竞拍</div>
            <button onClick={()=>setShowCreate(true)} style={{ padding:'12px 32px', background:'linear-gradient(135deg,#6366f1,#3b82f6)', color:'#fff', border:'none', borderRadius:12, fontSize:15, fontWeight:600, cursor:'pointer', transition:'all 0.2s' }} onMouseEnter={e=>{e.currentTarget.style.transform='scale(1.04)';e.currentTarget.style.boxShadow='0 4px 20px rgba(99,102,241,0.3)'}} onMouseLeave={e=>{e.currentTarget.style.transform='scale(1)';e.currentTarget.style.boxShadow='none'}}>+ 创建直播间</button>
          </div>
        )}

        {/* Stats — centered vertical, below room card */}
        <div style={{ display:'flex', flexDirection:'column', alignItems:'center', gap:10, marginTop:0, marginBottom:32 }}>
          {stats.map((s, i) => (
            <div key={s.label}
              onMouseEnter={() => setHoveredCard(s.label)}
              onMouseLeave={() => setHoveredCard(null)}
              onClick={() => s.link && navigate(s.link)}
              style={{
                background:hoveredCard===s.label?`linear-gradient(135deg, ${s.color}33, ${s.color}12)`:`linear-gradient(135deg, ${s.color}22, ${s.color}08)`,
                backdropFilter:'blur(12px)', borderRadius:16, padding:'16px 20px', width:'100%',
                border:hoveredCard===s.label?`1px solid ${s.color}55`:`1px solid ${s.color}22`,
                cursor: s.link ? 'pointer' : 'default',
                transition:'all 0.35s cubic-bezier(0.4, 0, 0.2, 1)',
                transform:hoveredCard===s.label?'translateY(-6px) scale(1.015)':'translateY(0) scale(1)',
                boxShadow:hoveredCard===s.label?`0 16px 40px ${s.color}22, 0 0 0 2px ${s.color}44, inset 0 1px 0 ${s.color}33`:'none',
                position:'relative', overflow:'hidden',
                animation:`cardIn 0.4s ease-out ${i*0.08}s both`,
              }}>
              {/* Shine sweep */}
              <div style={{
                position:'absolute', top:0, left:'-100%', width:'100%', height:'100%',
                background:`linear-gradient(105deg, transparent 40%, ${s.color}18 50%, transparent 60%)`,
                transition:'left 0.5s ease',
                ...(hoveredCard===s.label ? {left:'100%'} : {}),
              }} />
              {/* Top glow line */}
              <div style={{
                position:'absolute', top:0, left:'10%', right:'10%', height:1,
                background:hoveredCard===s.label?`linear-gradient(90deg, transparent, ${s.color}88, transparent)`:'transparent',
                transition:'all 0.3s ease',
              }} />
              <div style={{ fontSize:15, color:'rgba(148,163,184,0.5)', letterSpacing:1, marginBottom:6 }}>{s.label}</div>
              <div style={{ fontSize:32, fontWeight:700, color:'#e2e8f0' }}>{s.value}</div>
            </div>
          ))}
        </div>

      </div>

      {/* Modals (kept functional, dark themed) */}
      {showCreate && (
        <div style={{ position:'fixed', inset:0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex:100, backdropFilter:'blur(4px)' }}>
          <div style={{ background:'rgba(15,15,40,0.98)', backdropFilter:'blur(20px)', padding:28, borderRadius:20, width:400, border:'1px solid rgba(255,255,255,0.08)', animation:'fadeIn 0.3s ease-out' }}>
            <h2 style={{ marginTop:0, fontWeight:700, background:'linear-gradient(135deg,#e2e8f0,#a5b4fc)', WebkitBackgroundClip:'text', WebkitTextFillColor:'transparent' }}>创建直播间</h2>
            <input placeholder="直播间标题" value={title} onChange={e=>setTitle(e.target.value)} autoFocus style={{ display:'block', width:'100%', padding:'12px 16px', background:'rgba(255,255,255,0.04)', border:'1px solid rgba(255,255,255,0.08)', borderRadius:10, fontSize:15, color:'#e2e8f0', outline:'none', fontFamily:'inherit', boxSizing:'border-box', marginBottom:16 }} />
            <div style={{ display:'flex', gap:10 }}>
              <button onClick={handleCreate} style={{ flex:1, padding:12, background:'linear-gradient(135deg,#6366f1,#3b82f6)', color:'#fff', border:'none', borderRadius:10, cursor:'pointer', fontSize:14, fontWeight:600 }}>创建</button>
              <button onClick={()=>setShowCreate(false)} style={{ flex:1, padding:12, background:'rgba(255,255,255,0.06)', color:'rgba(226,232,240,0.6)', border:'1px solid rgba(255,255,255,0.08)', borderRadius:10, cursor:'pointer', fontSize:14 }}>取消</button>
            </div>
          </div>
        </div>
      )}

      {startRoom && (
        <div style={{ position:'fixed', inset:0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex:100, backdropFilter:'blur(4px)' }}>
          <div style={{ background:'rgba(15,15,40,0.98)', backdropFilter:'blur(20px)', padding:28, borderRadius:20, width:500, maxHeight:'80vh', overflow:'auto', border:'1px solid rgba(255,255,255,0.08)', animation:'fadeIn 0.3s ease-out' }}>
            <h2 style={{ marginTop:0, fontWeight:700, background:'linear-gradient(135deg,#e2e8f0,#a5b4fc)', WebkitBackgroundClip:'text', WebkitTextFillColor:'transparent' }}>🔴 开启直播</h2>
            <input placeholder="直播标题" value={startTitle} onChange={e=>setStartTitle(e.target.value)} autoFocus style={{ display:'block', width:'100%', padding:'12px 16px', background:'rgba(255,255,255,0.04)', border:'1px solid rgba(255,255,255,0.08)', borderRadius:10, fontSize:15, color:'#e2e8f0', outline:'none', fontFamily:'inherit', boxSizing:'border-box', marginBottom:14 }} />
            {/* Background video */}
            <div style={{ marginBottom:14 }}>
              <div style={{ fontSize:13, color:'rgba(148,163,184,0.5)', marginBottom:6 }}>直播背景视频</div>
              <label style={{
                display:'flex', alignItems:'center', gap:8,
                padding:'10px 14px', borderRadius:10, cursor:'pointer',
                background:'rgba(255,255,255,0.04)', border:'1px dashed rgba(255,255,255,0.1)',
                color: bgVideoUrl?'#34d399':'rgba(148,163,184,0.5)', fontSize:13, transition:'all 0.2s',
              }}
                onMouseEnter={e=>{e.currentTarget.style.borderColor='rgba(99,102,241,0.4)';e.currentTarget.style.color='#a5b4fc'}}
                onMouseLeave={e=>{e.currentTarget.style.borderColor='rgba(255,255,255,0.1)';e.currentTarget.style.color=bgVideoUrl?'#34d399':'rgba(148,163,184,0.5)'}}
              >
                <span style={{ fontSize:16 }}>🎬</span>
                {uploadingVideo ? '上传中...' : bgVideoUrl ? '✅ 已选择背景视频' : '点击上传背景视频（可选）'}
                <input type="file" accept="video/mp4,video/webm" onChange={handleVideoUpload} style={{ display:'none' }} />
              </label>
            </div>
            <div style={{ fontSize:14, fontWeight:600, marginBottom:10, color:'rgba(226,232,240,0.7)' }}>选择本场竞拍商品 ({selectedIds.size} 已选)</div>
            {loadingProducts ? <div style={{ textAlign:'center', padding:20, color:'rgba(148,163,184,0.4)' }}>加载商品中...</div>
            : selectProducts.length===0 ? <div style={{ textAlign:'center', padding:20, color:'rgba(148,163,184,0.4)' }}>暂无商品</div>
            : <div style={{ maxHeight:280, overflow:'auto', marginBottom:14 }}>
              {selectProducts.map((p:any)=>{
                const idx=[...selectedIds].indexOf(p.id); const sel=selectedIds.has(p.id);
                return <div key={p.id} onClick={()=>{const n=new Set(selectedIds);sel?n.delete(p.id):n.add(p.id);setSelectedIds(n)}} style={{ display:'flex', alignItems:'center', gap:8, padding:'10px 12px', marginBottom:4, borderRadius:10, cursor:'pointer', background:sel?'rgba(99,102,241,0.12)':'rgba(255,255,255,0.02)', border:sel?'2px solid rgba(99,102,241,0.4)':'2px solid transparent', transition:'all 0.15s' }}>
                  <span style={{ width:24, height:24, borderRadius:'50%', background:sel?'#6366f1':'rgba(255,255,255,0.06)', color:sel?'#fff':'rgba(148,163,184,0.4)', display:'flex', alignItems:'center', justifyContent:'center', fontSize:12, fontWeight:'bold', flexShrink:0 }}>{sel?(idx>=0?idx+1:'✓'):''}</span>
                  <div style={{ flex:1, fontSize:14, fontWeight:sel?'bold':'normal', color:sel?'#e2e8f0':'rgba(226,232,240,0.6)' }}>{p.title}</div>
                  <span style={{ fontSize:12, color:'rgba(148,163,184,0.4)' }}>起拍 ¥{fmt(p.start_price||'0')}</span>
                </div>
              })}
            </div>}
            <div style={{ display:'flex', gap:10 }}>
              <button onClick={handleStart} style={{ flex:1, padding:12, background:'linear-gradient(135deg,#10b981,#059669)', color:'#fff', border:'none', borderRadius:10, cursor:'pointer', fontSize:14, fontWeight:600 }}>开启直播</button>
              <button onClick={()=>{setStartRoom(null);setSelectedIds(new Set())}} style={{ flex:1, padding:12, background:'rgba(255,255,255,0.06)', color:'rgba(226,232,240,0.6)', border:'1px solid rgba(255,255,255,0.08)', borderRadius:10, cursor:'pointer', fontSize:14 }}>取消</button>
            </div>
          </div>
        </div>
      )}

      <style>{`
        @keyframes cardIn { from{opacity:0;transform:translateY(16px)} to{opacity:1;transform:translateY(0)} }
        @keyframes dashShimmer { 0%{left:-100%} 100%{left:200%} }
        @keyframes fadeIn { from{opacity:0} to{opacity:1} }
      `}</style>
    </div>
  );
}
