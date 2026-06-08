import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { useAuth } from '../../store/AuthContext';
import type { SellerOrderItem } from '../../types';

const fmt = (s?: string) =>
  s
    ? s.endsWith('.00')
      ? s.slice(0, -3)
      : s.includes('.')
        ? s.replace(/0+$/, '').replace(/\.$/, '')
        : s
    : '0';

const STATUS_LABELS: Record<string, string> = {
  '0': '待支付',
  '1': '已支付',
  '2': '已发货',
  '3': '已完成',
};

const STATUS_COLORS: Record<string, string> = {
  '0': '#f59e0b',
  '1': '#3b82f6',
  '2': '#10b981',
  '3': '#6366f1',
};

export default function SellerOrders() {
  const { token } = useAuth();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const statusParam = searchParams.get('status') || '';
  const title = statusParam === '0' ? '待支付订单' : statusParam === '3' ? '已完成订单' : '已售订单';

  // Default: fetch only paid+ orders for "已售", only unpaid for "待支付"
  const filterStatus = statusParam || '1,2,3';

  const [orders, setOrders] = useState<SellerOrderItem[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const pageSize = 10;

  useEffect(() => {
    if (!token) return;
    setLoading(true);
    const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
    params.set('status', filterStatus);

    fetch(`/api/seller/orders?${params.toString()}`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(r => r.json())
      .then(data => {
        if (data.code === 0) {
          setOrders(data.data.list || []);
          setTotal(data.data.total || 0);
        }
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [token, page, statusParam]);

  return (
    <div
      style={{
        minHeight: '100vh',
        background: 'linear-gradient(180deg,#0a0a1a 0%,#0d1b2a 50%,#0f172a 100%)',
        fontFamily: "'Noto Sans SC','PingFang SC',system-ui,sans-serif",
        color: '#e2e8f0',
      }}
    >
      <div style={{ position: 'fixed', top: '-10%', right: '-10%', width: '40vw', height: '40vw', background: 'radial-gradient(circle,rgba(99,102,241,0.06) 0%,transparent 70%)', borderRadius: '50%', pointerEvents: 'none' }} />

      <div style={{ maxWidth: 720, margin: '0 auto', padding: '32px 20px 60px', position: 'relative', zIndex: 1 }}>
        {/* Header */}
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 28, gap: 16 }}>
          <button
            onClick={() => navigate('/', { replace: true })}
            style={{
              background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)',
              borderRadius: 12, width: 40, height: 40, fontSize: 18, cursor: 'pointer',
              color: 'rgba(148,163,184,0.7)', display: 'flex', alignItems: 'center', justifyContent: 'center',
              transition: 'all 0.2s',
            }}
            onMouseEnter={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.08)'; e.currentTarget.style.color = '#fff'; }}
            onMouseLeave={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.04)'; e.currentTarget.style.color = 'rgba(148,163,184,0.7)'; }}
          >←</button>
          <h1 style={{
            margin: 0, fontSize: 24, fontWeight: 700,
            background: 'linear-gradient(135deg, #e2e8f0, #a5b4fc)',
            WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent',
          }}>{title} ({total})</h1>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: 60, color: 'rgba(148,163,184,0.4)', fontSize: 14 }}>加载中...</div>
        ) : orders.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 80, color: 'rgba(148,163,184,0.4)' }}>
            <div style={{ fontSize: 48, marginBottom: 12 }}>📭</div>
            <div style={{ fontSize: 15 }}>暂无订单</div>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {orders.map((order, i) => (
              <div key={order.id} style={{
                background: 'rgba(255,255,255,0.03)', backdropFilter: 'blur(12px)',
                borderRadius: 16, padding: '18px 20px',
                border: '1px solid rgba(255,255,255,0.06)',
                display: 'flex', alignItems: 'center', gap: 16,
                transition: 'transform 0.2s, box-shadow 0.2s',
                animation: `cardIn 0.4s ease-out ${i * 0.05}s both`,
              }}
                onMouseEnter={e => { e.currentTarget.style.transform = 'translateY(-2px)'; e.currentTarget.style.boxShadow = '0 8px 24px rgba(0,0,0,0.3)'; }}
                onMouseLeave={e => { e.currentTarget.style.transform = 'translateY(0)'; e.currentTarget.style.boxShadow = 'none'; }}
              >
                {/* Product image */}
                <div style={{
                  width: 64, height: 64, borderRadius: 12, flexShrink: 0,
                  background: order.product_image ? `url(${order.product_image})` : 'rgba(99,102,241,0.08)',
                  backgroundSize: 'cover', backgroundPosition: 'center',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  overflow: 'hidden', position: 'relative',
                }}>
                  {order.product_image ? (
                    <img src={order.product_image} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                      onError={e => { (e.target as HTMLImageElement).style.display = 'none'; }} />
                  ) : null}
                  {!order.product_image && <span style={{ fontSize: 24, color: 'rgba(148,163,184,0.3)', fontWeight: 'bold' }}>?</span>}
                </div>

                {/* Info */}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontWeight: 600, fontSize: 14, marginBottom: 6, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {order.product_title || '商品'}
                  </div>
                  <div style={{ fontSize: 12, color: 'rgba(148,163,184,0.5)', marginBottom: 4 }}>
                    订单号: {order.order_no}
                  </div>
                  <div style={{ fontSize: 20, fontWeight: 700, color: '#e53e3e' }}>
                    ¥{fmt(order.amount)}
                  </div>
                </div>

                {/* Buyer */}
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 4, flexShrink: 0, minWidth: 60 }}>
                  <img
                    src={order.buyer_avatar || undefined}
                    alt=""
                    style={{ width: 32, height: 32, borderRadius: '50%', background: '#667eea', border: '2px solid rgba(255,255,255,0.1)' }}
                    onError={e => { (e.target as HTMLImageElement).src = 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><circle cx="16" cy="16" r="16" fill="%23667eea"/><text x="16" y="22" text-anchor="middle" fill="white" font-size="14">' + (order.buyer_nickname?.[0] || '?') + '</text></svg>'; }}
                  />
                  <span style={{ fontSize: 11, color: 'rgba(226,232,240,0.6)', maxWidth: 64, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {order.buyer_nickname || '-'}
                  </span>
                </div>

                {/* Status badge + actions */}
                <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 6, flexShrink: 0 }}>
                  <span style={{
                    padding: '4px 10px', borderRadius: 20, fontSize: 11, fontWeight: 600,
                    color: STATUS_COLORS[order.status] || '#a0aec0',
                    background: `${STATUS_COLORS[order.status] || '#a0aec0'}18`,
                    border: `1px solid ${STATUS_COLORS[order.status] || '#a0aec0'}22`,
                  }}>
                    {STATUS_LABELS[order.status] || '未知'}
                  </span>
                  {order.status === 1 && (
                    <button onClick={async () => {
                      if (!token) return;
                      try {
                        await fetch(`/api/seller/orders/${order.id}/ship`, {
                          method: 'POST',
                          headers: { Authorization: `Bearer ${token}` },
                        });
                        setOrders(prev => prev.map(o => o.id === order.id ? { ...o, status: 2 } : o));
                      } catch { /* ignore */ }
                    }} style={{
                      background: 'linear-gradient(135deg, #3b82f6, #2563eb)', color: '#fff', border: 'none',
                      padding: '9px 24px', borderRadius: 10, fontSize: 14, fontWeight: 600,
                      cursor: 'pointer', letterSpacing: 0.5,
                      transition: 'transform 0.15s, box-shadow 0.15s',
                    }}
                      onMouseEnter={e => { e.currentTarget.style.transform = 'scale(1.04)'; e.currentTarget.style.boxShadow = '0 4px 16px rgba(59,130,246,0.3)'; }}
                      onMouseLeave={e => { e.currentTarget.style.transform = 'scale(1)'; e.currentTarget.style.boxShadow = 'none'; }}
                    >发货</button>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Pagination */}
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

      <style>{`
        @keyframes cardIn { from { opacity: 0; transform: translateY(16px); } to { opacity: 1; transform: translateY(0); } }
      `}</style>
    </div>
  );
}
