import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../../store/AuthContext';
import { seller as sellerApi } from '../../api';
import type { LiveRoom } from '../../types';

export default function SellerLiveRooms() {
  const { token } = useAuth();
  const navigate = useNavigate();
  const [rooms, setRooms] = useState<LiveRoom[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ title: '', cover_image: '', stream_url: '' });

  // Start modal state
  const [startRoom, setStartRoom] = useState<LiveRoom | null>(null);
  const [startTitle, setStartTitle] = useState('');

  const load = async () => {
    if (!token) return;
    try {
      const data = await sellerApi.listRooms(token);
      setRooms((data as any)?.list || data || []);
    } catch { /* ignore */ }
    setLoading(false);
  };

  useEffect(() => { load(); }, [token]);

  const handleCreate = async () => {
    if (!token || !form.title) return;
    try {
      await sellerApi.createRoom(form, token);
      setShowCreate(false);
      setForm({ title: '', cover_image: '', stream_url: '' });
      load();
    } catch (err: any) { alert(err.message); }
  };

  const handleStartClick = (room: LiveRoom) => {
    setStartRoom(room);
    setStartTitle(room.title || '');
  };

  const handleStartConfirm = async () => {
    if (!token || !startRoom || !startTitle.trim()) return;
    try {
      // Update title first
      await fetch(`/api/seller/live-rooms/${startRoom.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ title: startTitle.trim() }),
      });
      // Then start the stream
      await sellerApi.startLive(startRoom.id, token);
      setStartRoom(null);
      load();
      navigate(`/room/${startRoom.id}`);
    } catch (err: any) { alert(err.message); }
  };

  const handleEnd = async (id: number) => {
    if (!token) return;
    if (!confirm('确定结束直播？')) return;
    try {
      await sellerApi.endLive(id, token);
      load();
    } catch (err: any) { alert(err.message); }
  };

  const statusMap: Record<number, { label: string; color: string }> = {
    0: { label: '离线', color: '#718096' },
    1: { label: '直播中', color: '#e53e3e' },
    2: { label: '已结束', color: '#a0aec0' },
  };

  return (
    <div style={{ padding: 20, maxWidth: 1000, margin: '0 auto' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <h1 style={{ margin: 0 }}>📺 直播间管理 ({rooms.length})</h1>
        {rooms.length === 0 && (
          <button onClick={() => setShowCreate(true)}
            style={{ padding: '10px 20px', background: '#3182ce', color: '#fff', border: 'none', borderRadius: 8, cursor: 'pointer', fontSize: 14, fontWeight: 'bold' }}>
            + 创建直播间
          </button>
        )}
      </div>

      {loading ? (
        <div style={{ textAlign: 'center', padding: 40, color: '#a0aec0' }}>加载中...</div>
      ) : rooms.length === 0 ? (
        <div style={{ textAlign: 'center', padding: 60, color: '#a0aec0' }}>
          <div style={{ fontSize: 48 }}>📭</div>
          <div style={{ marginTop: 12 }}>暂无直播间，点击上方按钮创建</div>
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          {rooms.map(room => {
            const st = statusMap[room.status] || statusMap[2];
            return (
              <div key={room.id} style={{
                display: 'flex', alignItems: 'center', padding: '16px', background: '#fff',
                borderRadius: 10, boxShadow: '0 1px 3px rgba(0,0,0,0.08)',
              }}>
                <div style={{
                  width: 120, height: 68, borderRadius: 6, background: '#1a202c',
                  backgroundImage: room.cover_image ? `url(${room.cover_image})` : undefined,
                  backgroundSize: 'cover', backgroundPosition: 'center',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  marginRight: 16, flexShrink: 0,
                }}>
                  {!room.cover_image && <span style={{ fontSize: 28 }}>📺</span>}
                </div>
                <div style={{ flex: 1 }}>
                  <div style={{ fontWeight: 'bold', fontSize: 16, marginBottom: 4 }}>{room.title}</div>
                  <div style={{ fontSize: 13, color: '#718096' }}>
                    ID: {room.id} · 在线: {room.online_count || 0} 人 · 点赞: {room.total_likes || 0}
                  </div>
                </div>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <span style={{
                    padding: '4px 12px', borderRadius: 20, fontSize: 12, fontWeight: 'bold',
                    background: st.color === '#e53e3e' ? '#fff5f5' : '#f7fafc',
                    color: st.color, marginRight: 4,
                  }}>
                    {st.label}
                  </span>
                  {room.status === 1 && room.stream_url && (
                    <code style={{
                      fontSize: 11, color: '#6366f1', background: '#eef2ff',
                      padding: '3px 8px', borderRadius: 6, maxWidth: 240,
                      overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                    }} title={room.stream_url}>{room.stream_url}</code>
                  )}
                </div>
                <div style={{ display: 'flex', gap: 8 }}>
                  {(room.status === 0 || room.status === 2) && (
                    <button onClick={() => handleStartClick(room)}
                      style={btnStyle('#38a169')}>开始直播</button>
                  )}
                  {room.status === 1 && (
                    <button onClick={() => navigate(`/room/${room.id}`)}
                      style={btnStyle('#3182ce')}>进入直播间</button>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Create Modal */}
      {showCreate && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100 }}>
          <div style={{ background: '#fff', padding: 24, borderRadius: 12, width: 400 }}>
            <h2 style={{ marginTop: 0 }}>创建直播间</h2>
            <input placeholder="直播间标题 *" value={form.title}
              onChange={e => setForm({ ...form, title: e.target.value })}
              style={inputStyle} />
            <input placeholder="封面图片URL（可选）" value={form.cover_image}
              onChange={e => setForm({ ...form, cover_image: e.target.value })}
              style={inputStyle} />
            <input placeholder="推流地址（可选）" value={form.stream_url}
              onChange={e => setForm({ ...form, stream_url: e.target.value })}
              style={inputStyle} />
            <div style={{ display: 'flex', gap: 8, marginTop: 16 }}>
              <button onClick={handleCreate}
                style={{ flex: 1, padding: 10, background: '#3182ce', color: '#fff', border: 'none', borderRadius: 6, cursor: 'pointer', fontSize: 14 }}>
                创建
              </button>
              <button onClick={() => setShowCreate(false)}
                style={{ flex: 1, padding: 10, background: '#e2e8f0', border: 'none', borderRadius: 6, cursor: 'pointer', fontSize: 14 }}>
                取消
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Start Live Modal — ask for title */}
      {startRoom && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 100 }}>
          <div style={{ background: '#fff', padding: 24, borderRadius: 12, width: 400 }}>
            <h2 style={{ marginTop: 0 }}>开启直播</h2>
            <div style={{ fontSize: 14, color: '#718096', marginBottom: 12 }}>
              设置本次直播的标题，用户端可以看到
            </div>
            <input
              placeholder="直播标题 *"
              value={startTitle}
              onChange={e => setStartTitle(e.target.value)}
              autoFocus
              style={inputStyle}
            />
            <div style={{ display: 'flex', gap: 8, marginTop: 16 }}>
              <button onClick={handleStartConfirm}
                style={{ flex: 1, padding: 10, background: '#38a169', color: '#fff', border: 'none', borderRadius: 6, cursor: 'pointer', fontSize: 14 }}>
                开启直播
              </button>
              <button onClick={() => setStartRoom(null)}
                style={{ flex: 1, padding: 10, background: '#e2e8f0', border: 'none', borderRadius: 6, cursor: 'pointer', fontSize: 14 }}>
                取消
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

const inputStyle: React.CSSProperties = {
  display: 'block', width: '100%', marginTop: 10, padding: '10px 12px',
  border: '1px solid #e2e8f0', borderRadius: 6, fontSize: 14, boxSizing: 'border-box',
};

const btnStyle = (bg: string): React.CSSProperties => ({
  padding: '6px 14px', background: bg, color: '#fff', border: 'none',
  borderRadius: 6, cursor: 'pointer', fontSize: 13,
});
