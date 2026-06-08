import { useRef, useLayoutEffect } from 'react';
import type { RankItem } from '../types';

interface Props {
  ranking: RankItem[];
  myBid: RankItem | null;
  myUserId?: number;
}

export default function RankingList({ ranking, myBid, myUserId }: Props) {
  const listRef = useRef<HTMLDivElement>(null);
  const prevPosRef = useRef<Map<number, { top: number; rank: number }>>(new Map());

  useLayoutEffect(() => {
    if (!listRef.current) return;
    const container = listRef.current;
    const items = container.querySelectorAll<HTMLElement>('[data-user-id]');
    if (items.length === 0) return;

    // Capture NEW positions (React already updated the DOM)
    const newPositions = new Map<number, { top: number; rank: number }>();
    items.forEach((el) => {
      const uid = Number(el.dataset.userId);
      const item = ranking.find((r) => r.user_id === uid);
      newPositions.set(uid, { top: el.getBoundingClientRect().top, rank: item?.rank ?? 0 });
    });

    const oldPositions = prevPosRef.current;
    // First render — just store and return
    if (oldPositions.size === 0) {
      newPositions.forEach((v, k) => oldPositions.set(k, v));
      return;
    }

    // Check if any rank actually changed
    let changed = false;
    for (const [uid, pos] of newPositions) {
      const old = oldPositions.get(uid);
      if (!old || old.rank !== pos.rank) {
        changed = true;
        break;
      }
    }
    if (!changed) {
      // Update stored positions (scroll may have shifted them) but don't animate
      newPositions.forEach((v, k) => oldPositions.set(k, v));
      return;
    }

    // FLIP: Invert
    items.forEach((el) => {
      const uid = Number(el.dataset.userId);
      const old = oldPositions.get(uid);
      const cur = newPositions.get(uid);
      if (!old || !cur) return;
      const dy = old.top - cur.top;
      if (Math.abs(dy) < 1) return;
      el.style.transition = 'none';
      el.style.transform = `translateY(${dy}px)`;
      // Items moving UP (old rank > new rank → lower position → positive dy) go on top
      el.style.zIndex = dy > 0 ? '3' : '1';
    });

    // Force layout, then play
    void container.offsetHeight;

    requestAnimationFrame(() => {
      items.forEach((el) => {
        el.style.transition = 'transform 0.45s cubic-bezier(0.25, 0.46, 0.45, 0.94)';
        el.style.transform = 'translateY(0)';
      });

      setTimeout(() => {
        items.forEach((el) => {
          el.style.transition = '';
          el.style.transform = '';
          el.style.zIndex = '';
        });
      }, 470);
    });

    // Save new positions for next comparison
    oldPositions.clear();
    newPositions.forEach((v, k) => oldPositions.set(k, v));
  }, [ranking]);

  if (ranking.length === 0) {
    return (
      <div style={{ color: '#718096', padding: '20px', textAlign: 'center', fontSize: 13 }}>
        暂无出价，快来抢沙发！
      </div>
    );
  }

  return (
    <div ref={listRef}>
      {ranking.map((item) => {
        const isMe = myUserId != null && item.user_id === myUserId;
        return (
          <div
            key={item.user_id}
            data-user-id={item.user_id}
            style={{
              display: 'flex', alignItems: 'center', padding: '8px 12px',
              background: isMe ? 'rgba(49,130,206,0.15)' : 'transparent',
              borderBottom: '1px solid #1e1e3a',
              fontWeight: isMe ? 'bold' : 'normal',
              color: '#e2e8f0',
              position: 'relative',
            }}
          >
            <span style={{ width: 30, color: item.rank <= 3 ? '#fc8181' : '#718096', fontWeight: 'bold', flexShrink: 0 }}>
              {item.rank <= 3 ? ['🥇', '🥈', '🥉'][item.rank - 1] : `#${item.rank}`}
            </span>
            <img
              src={item.avatar || '/default-avatar.png'}
              alt=""
              style={{ width: 28, height: 28, borderRadius: '50%', marginRight: 8, flexShrink: 0 }}
              onError={(e) => { (e.target as HTMLImageElement).src = 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><circle cx="16" cy="16" r="16" fill="%234a5568"/></svg>'; }}
            />
            <span style={{ flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {item.nickname || `用户${item.user_id}`}
            </span>
            <span style={{ color: '#fc8181', fontWeight: 'bold', flexShrink: 0 }}>¥{item.amount}</span>
          </div>
        );
      })}

      {myBid && !ranking.find(r => r.user_id === myUserId) && (
        <div style={{
          display: 'flex', alignItems: 'center', padding: '8px 12px',
          background: 'rgba(49,130,206,0.2)', borderTop: '2px solid #3182ce',
          marginTop: 8, fontWeight: 'bold', color: '#e2e8f0',
        }}>
          <span style={{ width: 30, color: '#a0aec0' }}>#{myBid.rank}</span>
          <span style={{ flex: 1 }}>我</span>
          <span style={{ color: '#fc8181' }}>¥{myBid.amount}</span>
        </div>
      )}
    </div>
  );
}
