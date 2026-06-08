import { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../../store/AuthContext';
import { buyer as buyerApi } from '../../api';
import type { SellerOrderItem } from '../../types';

const fmt = (s?: string) =>
  s
    ? s.endsWith('.00')
      ? s.slice(0, -3)
      : s.includes('.')
        ? s.replace(/0+$/, '').replace(/\.$/, '')
        : s
    : '0';

const STATUS_STYLE: Record<number, { label: string; dot: string; bar: string }> = {
  0: { label: '待支付', dot: '#f59e0b', bar: 'rgba(245,158,11,0.15)' },
  1: { label: '已支付', dot: '#3b82f6', bar: 'rgba(59,130,246,0.15)' },
  2: { label: '已发货', dot: '#10b981', bar: 'rgba(16,185,129,0.15)' },
  3: { label: '已完成', dot: '#6366f1', bar: 'rgba(99,102,241,0.15)' },
};

const PAY_DEADLINE_SEC = 30 * 60; // 30 minutes

export default function Orders() {
  const { token } = useAuth();
  const navigate = useNavigate();
  const [orders, setOrders] = useState<SellerOrderItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [tick, setTick] = useState(0);
  const [expiredModal, setExpiredModal] = useState(false);
  const baseTimeRef = useRef(Date.now());

  // Tick every second for countdown
  useEffect(() => {
    const t = setInterval(() => setTick(x => x + 1), 1000);
    return () => clearInterval(t);
  }, []);

  useEffect(() => {
    if (!token) return;
    buyerApi.myOrders(token)
      .then((data: any) => {
        setOrders(data.list || data || []);
        baseTimeRef.current = Date.now();
      })
      .catch((err: any) => setError(err.message))
      .finally(() => setLoading(false));
  }, [token]);

  // Compute live remaining seconds
  const liveRemaining = (order: SellerOrderItem): number => {
    if (order.status !== 0) return 0;
    const elapsed = Math.floor((Date.now() - baseTimeRef.current) / 1000);
    return Math.max(0, order.remaining_sec - elapsed);
  };

  const handlePay = async (order: SellerOrderItem) => {
    if (!token) return;
    if (liveRemaining(order) <= 0) {
      setExpiredModal(true);
      return;
    }
    try {
      const result: any = await buyerApi.payOrder(order.id, token);
      if (result?.status) {
        setOrders(prev =>
          prev.map(o => o.id === order.id ? { ...o, status: result.status } : o)
        );
      }
    } catch (err: any) {
      const msg = err.message || '';
      if (msg.includes('expired') || msg.includes('deadline') || msg.includes('过期')) {
        setExpiredModal(true);
        return;
      }
      alert(msg || '支付失败');
    }
  };

  return (
    <div style={{
      minHeight: '100vh',
      background: 'linear-gradient(180deg, #0a0a1a 0%, #0d1b2a 50%, #0f172a 100%)',
      fontFamily: "'Noto Sans SC', 'PingFang SC', system-ui, sans-serif",
      color: '#e2e8f0',
    }}>
      {/* Ambient */}
      <div style={{ position: 'fixed', top: '-10%', right: '-10%', width: '40vw', height: '40vw', background: 'radial-gradient(circle, rgba(99,102,241,0.06) 0%, transparent 70%)', borderRadius: '50%', pointerEvents: 'none' }} />

      <div style={{ maxWidth: 680, margin: '0 auto', padding: '32px 20px 60px', position: 'relative', zIndex: 1 }}>
        {/* Header */}
        <div style={{ display: 'flex', alignItems: 'center', marginBottom: 32, gap: 16 }}>
          <button onClick={() => navigate('/', { replace: true })} style={{
            background: 'rgba(255,255,255,0.04)', border: '1px solid rgba(255,255,255,0.08)',
            borderRadius: 12, width: 40, height: 40, fontSize: 18, cursor: 'pointer',
            color: 'rgba(148,163,184,0.7)', display: 'flex', alignItems: 'center', justifyContent: 'center',
            transition: 'all 0.2s',
          }}
            onMouseEnter={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.08)'; e.currentTarget.style.color = '#fff'; }}
            onMouseLeave={e => { e.currentTarget.style.background = 'rgba(255,255,255,0.04)'; e.currentTarget.style.color = 'rgba(148,163,184,0.7)'; }}
          >←</button>
          <h1 style={{
            margin: 0, fontSize: 26, fontWeight: 700,
            background: 'linear-gradient(135deg, #e2e8f0, #a5b4fc)', WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent',
          }}>我的订单</h1>
        </div>

        {loading ? (
          <div style={{ textAlign: 'center', padding: 60, color: 'rgba(148,163,184,0.4)', fontSize: 14 }}>加载中...</div>
        ) : error ? (
          <div style={{ textAlign: 'center', padding: 40, color: '#fca5a5', background: 'rgba(239,68,68,0.08)', borderRadius: 12, border: '1px solid rgba(239,68,68,0.12)' }}>{error}</div>
        ) : orders.length === 0 ? (
          <div style={{ textAlign: 'center', padding: 80, color: 'rgba(148,163,184,0.4)' }}>
            <div style={{ fontSize: 56, marginBottom: 16 }}>📭</div>
            <div style={{ fontSize: 15 }}>暂无订单</div>
          </div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
            {orders.map((order, i) => {
              const st = STATUS_STYLE[order.status] || STATUS_STYLE[3];
              return (
                <div key={order.id} style={{
                  background: 'rgba(255,255,255,0.03)', backdropFilter: 'blur(12px)',
                  borderRadius: 16, padding: '20px 22px',
                  border: '1px solid rgba(255,255,255,0.06)',
                  transition: 'transform 0.2s, box-shadow 0.2s',
                  animation: `cardIn 0.4s ease-out ${i * 0.06}s both`,
                  position: 'relative', overflow: 'hidden',
                }}
                  onMouseEnter={e => { e.currentTarget.style.transform = 'translateY(-2px)'; e.currentTarget.style.boxShadow = '0 8px 24px rgba(0,0,0,0.3)'; }}
                  onMouseLeave={e => { e.currentTarget.style.transform = 'translateY(0)'; e.currentTarget.style.boxShadow = 'none'; }}
                >
                  {/* Left colored bar */}
                  <div style={{ position: 'absolute', left: 0, top: 0, bottom: 0, width: 3, background: st.bar }} />

                  {/* Product: image + name */}
                  <div style={{ display: 'flex', alignItems: 'center', gap: 14, marginBottom: 14 }}>
                    <div style={{
                      width: 52, height: 52, borderRadius: 10, flexShrink: 0,
                      background: order.product_image ? `url(${order.product_image})` : 'rgba(99,102,241,0.08)',
                      backgroundSize: 'cover', backgroundPosition: 'center',
                      display: 'flex', alignItems: 'center', justifyContent: 'center',
                      overflow: 'hidden',
                    }}>
                      {order.product_image ? (
                        <img src={order.product_image} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }}
                          onError={e => { (e.target as HTMLImageElement).style.display = 'none'; }} />
                      ) : null}
                      {!order.product_image && <span style={{ fontSize: 20, color: 'rgba(148,163,184,0.3)' }}>📦</span>}
                    </div>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontWeight: 600, fontSize: 14, color: '#e2e8f0', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {order.product_title || '商品'}
                      </div>
                      <div style={{ fontSize: 11, color: 'rgba(148,163,184,0.6)', marginTop: 3, letterSpacing: 0.5 }}>
                        订单号 {order.order_no}
                      </div>
                    </div>
                    <span style={{
                      display: 'inline-flex', alignItems: 'center', gap: 6,
                      padding: '5px 14px', borderRadius: 20, fontSize: 12, fontWeight: 600,
                      background: st.bar, color: st.dot, flexShrink: 0,
                    }}>
                      <span style={{ width: 6, height: 6, borderRadius: '50%', background: st.dot }} />
                      {st.label}
                    </span>
                  </div>

                  {/* Bottom row: amount + action */}
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end' }}>
                    <div>
                      <div style={{ fontSize: 11, color: 'rgba(148,163,184,0.6)', marginBottom: 3 }}>成交金额</div>
                      <div style={{ fontSize: 22, fontWeight: 700, color: '#e2e8f0' }}>
                        ¥{fmt(order.amount)}
                      </div>
                    </div>

                    <div style={{ textAlign: 'right' }}>
                      {order.status === 0 && (() => {
                          const rem = liveRemaining(order);
                          return (
                        <div>
                          {rem > 0 && (
                            <div style={{ fontSize: 12, color: rem < 60 ? '#ef4444' : '#f59e0b', marginBottom: 8, fontWeight: 500 }}>
                              ⏱ {Math.floor(rem / 60)} 分 {rem % 60} 秒
                            </div>
                          )}
                          {rem <= 0 && (
                            <div style={{ fontSize: 12, color: '#ef4444', marginBottom: 8, fontWeight: 500 }}>
                              ⏱ 已超时
                            </div>
                          )}
                          <button onClick={() => handlePay(order)} style={{
                            background: 'linear-gradient(135deg, #f59e0b, #d97706)', color: '#fff', border: 'none',
                            padding: '9px 24px', borderRadius: 10, fontSize: 14, fontWeight: 600,
                            cursor: 'pointer', letterSpacing: 0.5,
                            transition: 'transform 0.15s, box-shadow 0.15s',
                          }}
                            onMouseEnter={e => { e.currentTarget.style.transform = 'scale(1.04)'; e.currentTarget.style.boxShadow = '0 4px 16px rgba(245,158,11,0.3)'; }}
                            onMouseLeave={e => { e.currentTarget.style.transform = 'scale(1)'; e.currentTarget.style.boxShadow = 'none'; }}
                          >立即支付</button>
                        </div>
                      ); })()}
                      {order.status === 1 && <div style={{ fontSize: 13, color: 'rgba(148,163,184,0.5)' }}>等待卖家发货</div>}
                      {order.status === 2 && (
                        <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', gap: 6 }}>
                          <div style={{ fontSize: 13, color: '#10b981' }}>卖家已发货</div>
                          <button onClick={async () => {
                            if (!token) return;
                            try {
                              await fetch(`/api/orders/${order.id}/confirm`, {
                                method: 'POST',
                                headers: { Authorization: `Bearer ${token}` },
                              });
                              setOrders(prev => prev.map(o => o.id === order.id ? { ...o, status: 3 } : o));
                            } catch { /* ignore */ }
                          }} style={{
                            background: 'linear-gradient(135deg, #10b981, #059669)', color: '#fff', border: 'none',
                            padding: '9px 24px', borderRadius: 10, fontSize: 14, fontWeight: 600,
                            cursor: 'pointer', letterSpacing: 0.5,
                            transition: 'transform 0.15s, box-shadow 0.15s',
                          }}
                            onMouseEnter={e => { e.currentTarget.style.transform = 'scale(1.04)'; e.currentTarget.style.boxShadow = '0 4px 16px rgba(16,185,129,0.3)'; }}
                            onMouseLeave={e => { e.currentTarget.style.transform = 'scale(1)'; e.currentTarget.style.boxShadow = 'none'; }}
                          >确认收货</button>
                        </div>
                      )}
                      {order.status === 3 && <div style={{ fontSize: 13, color: 'rgba(148,163,184,0.4)' }}>交易完成</div>}
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Expired order modal */}
      {expiredModal && (
        <div style={{
          position: 'fixed', inset: 0, zIndex: 500,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(4px)',
        }} onClick={() => setExpiredModal(false)}>
          <div style={{
            background: 'rgba(15,15,40,0.98)', backdropFilter: 'blur(20px)',
            borderRadius: 16, padding: '32px 36px',
            border: '1px solid rgba(255,255,255,0.08)',
            textAlign: 'center', maxWidth: 340,
            animation: 'cardIn 0.3s ease-out',
          }} onClick={e => e.stopPropagation()}>
            <div style={{ fontSize: 22, fontWeight: 700, color: '#e2e8f0', marginBottom: 12 }}>
              订单已过期
            </div>
            <div style={{ fontSize: 13, color: 'rgba(148,163,184,0.5)', lineHeight: 1.6 }}>
              该订单已超过支付时效，请重新参与竞拍
            </div>
            <button onClick={() => setExpiredModal(false)} style={{
              marginTop: 20, padding: '10px 40px',
              background: 'linear-gradient(135deg, #6366f1, #3b82f6)',
              color: '#fff', border: 'none', borderRadius: 10,
              fontSize: 14, fontWeight: 600, cursor: 'pointer',
            }}>知道了</button>
          </div>
        </div>
      )}

      <style>{`@keyframes cardIn { from { opacity: 0; transform: translateY(16px); } to { opacity: 1; transform: translateY(0); } }`}</style>
    </div>
  );
}
