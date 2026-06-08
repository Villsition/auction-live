import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../../store/AuthContext';

interface BidHistoryItem {
  bid_id: number;
  auction_id: number;
  bid_amount: string;
  bid_time: string;
  product_id: number;
  product_title: string;
  product_image: string;
  final_price: string | null;
  auction_status: number;
  winner_id: number | null;
  room_id: number;
  seller_nickname: string;
  seller_avatar: string;
}

const fmt = (s?: string) =>
  s ? (s.endsWith('.00') ? s.slice(0, -3) : s.includes('.') ? s.replace(/0+$/, '').replace(/\.$/, '') : s) : '0';

export default function BidHistory() {
  const { token, user } = useAuth();
  const navigate = useNavigate();
  const [items, setItems] = useState<BidHistoryItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const pageSize = 10;

  useEffect(() => {
    if (!token) return;
    setLoading(true);
    fetch(`/api/bids/history?page=${page}&page_size=${pageSize}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(r => r.json())
      .then(data => {
        if (data.code === 0) {
          setItems(data.data.list || []);
          setTotal(data.data.total || 0);
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [token, page]);

  return (
    <div style={{
      minHeight: '100vh',
      background: 'linear-gradient(180deg, #0a0a1a 0%, #0d1b2a 50%, #0f172a 100%)',
      fontFamily: "'Noto Sans SC', 'PingFang SC', system-ui, sans-serif",
      color: '#e2e8f0',
    }}>
      <div style={{ position: 'fixed', top: '-10%', right: '-10%', width: '40vw', height: '40vw', background: 'radial-gradient(circle, rgba(99,102,241,0.06) 0%, transparent 70%)', borderRadius: '50%', pointerEvents: 'none' }} />

      <div style={{ maxWidth: 720, margin: '0 auto', padding: '32px 20px 60px', position: 'relative', zIndex: 1 }}>
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 28, gap: 16 }}>
          <button onClick={() => navigate('/', { replace: true })} style={{
            background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)',
            borderRadius: 12, width: 40, height: 40, fontSize: 18, cursor: 'pointer',
            color: 'rgba(148,163,184,0.7)', display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}>←</button>
          <h1 style={{
            margin: 0, fontSize: 24, fontWeight: 700,
            background: 'linear-gradient(135deg, #e2e8f0, #a5b4fc)',
            WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent',
          }}>历史竞拍记录 ({total})</h1>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: 60, color: 'rgba(148,163,184,0.4)', fontSize: 14 }}>加载中...</div>
        ) : items.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 80, color: 'rgba(148,163,184,0.4)' }}>
            <div style={{ fontSize: 48, marginBottom: 12 }}>📭</div>
            <div style={{ fontSize: 15 }}>暂无竞拍记录</div>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {items.map((item, i) => {
              const isWin = user && item.winner_id === user.id;
              return (
                <div key={item.bid_id} style={{
                  background: 'rgba(255,255,255,0.03)', backdropFilter: 'blur(12px)',
                  borderRadius: 16, padding: '18px 20px',
                  border: '1px solid rgba(255,255,255,0.06)',
                  display: 'flex', alignItems: 'center', gap: 14,
                  transition: 'transform 0.2s',
                  animation: `cardIn 0.4s ease-out ${i * 0.05}s both`,
                }}
                  onMouseEnter={e => { e.currentTarget.style.transform = 'translateY(-2px)'; }}
                  onMouseLeave={e => { e.currentTarget.style.transform = 'translateY(0)'; }}
                >
                  {/* Product image */}
                  <div style={{
                    width: 52, height: 52, borderRadius: 10, flexShrink: 0,
                    background: item.product_image ? `url(${item.product_image})` : 'rgba(99,102,241,0.08)',
                    backgroundSize: 'cover', backgroundPosition: 'center',
                    display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden',
                  }}>
                    {item.product_image ? (
                      <img src={item.product_image} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                        onError={e => { (e.target as HTMLImageElement).style.display = 'none'; }} />
                    ) : <span style={{ fontSize: 20, color: 'rgba(148,163,184,0.3)' }}>📦</span>}
                  </div>

                  {/* Info */}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ fontWeight: 600, fontSize: 14, marginBottom: 4, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {item.product_title || '商品'}
                    </div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 4 }}>
                      <span style={{ fontSize: 11, color: 'rgba(148,163,184,0.35)', marginRight: 2 }}>所在直播间：</span>
                      <img
                        src={item.seller_avatar || undefined} alt=""
                        style={{ width: 18, height: 18, borderRadius: '50%', background: '#667eea' }}
                        onError={e => { (e.target as HTMLImageElement).src = 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><circle cx="16" cy="16" r="16" fill="%23667eea"/><text x="16" y="22" text-anchor="middle" fill="white" font-size="12">' + (item.seller_nickname?.[0] || '?') + '</text></svg>'; }}
                      />
                      <span style={{ fontSize: 12, color: 'rgba(226,232,240,0.7)' }}>{item.seller_nickname || '-'}</span>
                    </div>
                    <div style={{ fontSize: 12, color: 'rgba(148,163,184,0.4)' }}>
                      我的出价 ¥{fmt(item.bid_amount)}
                      {item.final_price && ` · 落槌价 ¥${fmt(item.final_price)}`}
                    </div>
                  </div>

                  {/* Win badge */}
                  <div style={{ flexShrink: 0 }}>
                    {isWin ? (
                      <span style={{
                        padding: '6px 14px', borderRadius: 20, fontSize: 12, fontWeight: 700,
                        background: 'linear-gradient(135deg, rgba(245,158,11,0.2), rgba(251,191,36,0.1))',
                        color: '#fbbf24', border: '1px solid rgba(251,191,36,0.3)',
                        letterSpacing: 0.5,
                      }}>🏆 成功拍得</span>
                    ) : (
                      <span style={{
                        padding: '4px 12px', borderRadius: 20, fontSize: 11, fontWeight: 500,
                        color: 'rgba(148,163,184,0.4)',
                      }}>未拍得</span>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {total > pageSize && (
          <div style={{ textAlign: 'center', marginTop: 24 }}>
            <button disabled={page <= 1} onClick={() => setPage(p => p - 1)} style={{
              padding: '8px 18px', background: 'rgba(255,255,255,0.04)', color: 'rgba(226,232,240,0.6)',
              border: '1px solid rgba(255,255,255,0.06)', borderRadius: 8, cursor: 'pointer', fontSize: 13,
              opacity: page <= 1 ? 0.3 : 1,
            }}>上一页</button>
            <span style={{ margin: '0 14px', fontSize: 14, color: 'rgba(148,163,184,0.5)' }}>
              {page}/{Math.ceil(total / pageSize)}
            </span>
            <button disabled={page >= Math.ceil(total / pageSize)} onClick={() => setPage(p => p + 1)} style={{
              padding: '8px 18px', background: 'rgba(255,255,255,0.04)', color: 'rgba(226,232,240,0.6)',
              border: '1px solid rgba(255,255,255,0.06)', borderRadius: 8, cursor: 'pointer', fontSize: 13,
              opacity: page >= Math.ceil(total / pageSize) ? 0.3 : 1,
            }}>下一页</button>
          </div>
        )}
      </div>

      <style>{`@keyframes cardIn { from { opacity: 0; transform: translateY(16px); } to { opacity: 1; transform: translateY(0); } }`}</style>
    </div>
  );
}
