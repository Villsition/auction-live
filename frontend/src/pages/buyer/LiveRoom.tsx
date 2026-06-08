import { useState, useEffect, useCallback, useRef } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../../store/AuthContext';
import { useWebSocket } from '../../hooks/useWebSocket';
import { publicApi, buyer as buyerApi } from '../../api';
import Countdown from '../../components/Countdown';
import RankingList from '../../components/RankingList';
import type { RoomAuction, RankItem, WSBidEvent, WSAuctionEvent, WSOutbidEvent } from '../../types';

interface Comment {
  id: number;
  room_id: number;
  user_id: number;
  content: string;
  created_at: string;
  username?: string;
}

export default function BuyerLiveRoom() {
  const { roomId } = useParams<{ roomId: string }>();
  const { user, token } = useAuth();
  const navigate = useNavigate();
  const rid = Number(roomId);
  const isSeller = user && user.role >= 1 && user.id !== 0;

  const [data, setData] = useState<RoomAuction | null>(null);
  const [ranking, setRanking] = useState<RankItem[]>([]);
  const [myBid, setMyBid] = useState<RankItem | null>(null);
  const [amount, setAmount] = useState('');
  const [msg, setMsg] = useState('');
  const [toast, setToast] = useState('');
  const [polling, setPolling] = useState(true);
  const [showCart, setShowCart] = useState(false);
  const [cartClosing, setCartClosing] = useState(false);
  const [products, setProducts] = useState<any[]>([]);

  // Comments state
  const [comments, setComments] = useState<Comment[]>([]);
  const [commentText, setCommentText] = useState('');
  const commentListRef = useRef<HTMLDivElement>(null);
  const seenCommentIds = useRef<Set<number>>(new Set());

  const scrollCommentsToBottom = () => {
    setTimeout(() => {
      if (commentListRef.current) {
        commentListRef.current.scrollTop = commentListRef.current.scrollHeight;
      }
    }, 100);
  };

  const { connected, subscribe } = useWebSocket(rid, token);

  const [onlineCount, setOnlineCount] = useState(0);
  const [viewers, setViewers] = useState<{user_id: number; nickname: string; avatar: string}[]>([]);
  const [error, setError] = useState('');

  const API_HOST = '';
  const fmt = (s?: string) => (s ? (s.endsWith('.00') ? s.slice(0, -3) : s.includes('.') ? s.replace(/0+$/, '').replace(/\.$/, '') : s) : '0');

  // Load auction data
  const loadAuction = useCallback(async () => {
    try {
      const d = await publicApi.roomAuction(rid);
      setData(d);
      setOnlineCount(d.live_room?.online_count || 0);
      // If auction already ended, capture the ended product for the card
      if (d.auction_session && d.auction_session.status >= 2 && d.product) {
        showEndedProduct({...d.product, status: d.auction_session.status});
      }
      if (d.auction_session) {
        const cp = d.current_price || d.auction_session.current_price || '0';
        const inc = d.auction_session.bid_increment || '10';
        setAmount(String(Number(cp) + Number(inc)));
        if (d.auction_session.status !== 1) setPolling(false);
      }
      setError('');
    } catch (e: any) {
      setError(e.message || '加载失败');
    }
  }, [rid]);

  // Load ranking
  const loadRanking = useCallback(async () => {
    if (!data?.auction_session) return;
    try {
      const r = await publicApi.bidRanking(data.auction_session.id, 20);
      setRanking(r.ranking);
      setMyBid(r.my_bid);
    } catch { /* ignore */ }
  }, [data?.auction_session]);

  // Load comments
  const loadComments = useCallback(async () => {
    try {
      const res = await fetch(`${API_HOST}/api/live-rooms/${rid}/comments?limit=100`);
      const json = await res.json();
      if (json.code === 0) {
        const list: Comment[] = json.data || [];
        list.forEach(c => seenCommentIds.current.add(c.id));
        setComments(list);
        scrollCommentsToBottom();
      }
    } catch { /* ignore */ }
  }, [rid]);

  // Inject slide-up animation keyframes (no cleanup — keep forever)
  useEffect(() => {
    if (document.getElementById('cmt-slide-css')) return;
    const style = document.createElement('style');
    style.id = 'cmt-slide-css';
    style.textContent = `@keyframes cmtUp{from{opacity:0;transform:translateY(24px)}to{opacity:1;transform:translateY(0)}}.cmt-slide{animation:cmtUp .5s ease-out both}`;
    document.head.appendChild(style);
  }, []);

  // Initial load — reset comment tracking when room changes
  useEffect(() => {
    loadAuction();
    loadProducts();
    seenCommentIds.current.clear();
  }, [loadAuction]);

  // Poll ranking every 3s — keep last data when polling stops
  useEffect(() => {
    if (!polling) return;
    loadRankingRef.current();
    const t = setInterval(loadRanking, 3000);
    return () => clearInterval(t);
  }, [polling, loadRanking]);

  // Save ranking snapshot
  const rankingRef = useRef(ranking);
  rankingRef.current = ranking;
  const rankingGuardRef = useRef(ranking);
  const myBidRef = useRef(myBid);
  myBidRef.current = myBid;
  const loadRankingRef = useRef(loadRanking);
  loadRankingRef.current = loadRanking;
  // Refs for WS handlers — always current, never stale
  const dataRef = useRef(data);
  dataRef.current = data;
  const productsRef = useRef(products);
  productsRef.current = products;
  const userRef = useRef(user);
  userRef.current = user;
  const tokenRef = useRef(token);
  tokenRef.current = token;

  // Poll comments every 5s
  useEffect(() => {
    loadComments();
    const t = setInterval(loadComments, 5000);
    return () => clearInterval(t);
  }, [loadComments]);

  // WebSocket events — subscriptions are permanent (stable deps), all reads go through refs
  useEffect(() => {
    const unsubs = [
      subscribe('delay_extend', (d: any) => {
        const cur = dataRef.current;
        if (d.auction_id === cur?.auction_session?.id && d.new_end_time_ms > 0) {
          const inc = cur?.auction_session?.bid_increment || '10';
          setData(prev => prev ? { ...prev, end_timestamp_ms: d.new_end_time_ms, current_price: d.amount, bid_count: d.bid_count } : prev);
          setAmount(String(Number(d.amount) + Number(inc)));
          const sec = cur?.auction_session?.delay_seconds || 30;
          showCenterMsg(`+${sec}s`, 'delay');
        }
      }),
      ...[
        (d: any) => {
          const e = d as WSBidEvent;
          const cur = dataRef.current;
          if (e.auction_id === cur?.auction_session?.id) {
            if (e.ceiling_deal) {
              setData(prev => prev ? {
                ...prev,
                current_price: e.amount,
                bid_count: e.bid_count,
                auction_session: prev.auction_session ? { ...prev.auction_session, status: 2 } : prev.auction_session,
              } : prev);
              setToast('🔨 封顶价成交！');
            } else {
              setData(prev => prev ? { ...prev, current_price: e.amount, bid_count: e.bid_count } : prev);
              const inc = cur?.auction_session?.bid_increment || '10';
              setAmount(String(Number(e.amount) + Number(inc)));
            }
            // Update bid_count in showcase
            setProducts(prev => prev.map(p =>
              p.auction_id === e.auction_id ? { ...p, bid_count: e.bid_count } : p
            ));
            loadRankingRef.current();
          }
        },
      ].flatMap(fn => [subscribe('bid', fn), subscribe('ceiling_deal', fn)]),
      subscribe('auction_start', () => {
        loadAuction();
        loadProducts();
      }),
      subscribe('auction_cancel', (d: any) => {
        loadAuction();
        loadProducts();
        startEndedCountdown();
        const cp = productsRef.current.find((p: any) => p.auction_id === d.auction_id);
        if (cp) showEndedProduct({...cp, status: 4});
      }),
      subscribe('auction_end', (d) => {
        const e = d as WSAuctionEvent;
        const cur = dataRef.current;
        if (e.auction_id === cur?.auction_session?.id) {
          setPolling(false);
          if (cur?.product) showEndedProduct({...cur.product, status: 2});
          startEndedCountdown();
          if (rankingRef.current.length > 0) setRanking(rankingRef.current);

          openEndedModal(e.winner_id, '', '', e.final_price);

          const uid = userRef.current?.id;
          const tok = tokenRef.current;
          if (e.winner_id === uid && tok) {
            setTimeout(async () => {
              try {
                const resp = await fetch(`/api/orders?token=${tok}`, {headers:{Authorization:`Bearer ${tok}`}});
                const orders = await resp.json();
                const orderList = orders.data?.list || orders.data || [];
                const latest = orderList[0];
                if (latest) {
                  setEndedModalData(prev => prev ? { ...prev, orderId: latest.id, expireSec: latest.remaining_sec } : prev);
                }
              } catch { /* ignore */ }
            }, 1500);
          }
          loadAuction();
          loadProducts();
        }
      }),
      subscribe('new_leader', (d: any) => {
        loadRankingRef.current();
        if (d.new_leader_id === userRef.current?.id) showCenterMsg('🎉 领先！', 'lead');
      }),
      subscribe('outbid', (d) => {
        const e = d as WSOutbidEvent;
        if (e.user_id === userRef.current?.id) {
          showCenterMsg('⚡ 被超越！', 'outbid');
          loadRankingRef.current();
        }
      }),
      subscribe('online_count', (d: any) => {
        if (d.room_id === rid) {
          setOnlineCount(d.count);
          if (d.viewers) setViewers(d.viewers);
        }
      }),
      subscribe('like', (d: any) => {
        if (d.room_id === rid) setLikeCount(d.total || (d as any).count || 0);
      }),
      subscribe('live_end', () => {
        setLiveEnded(true);
        setPolling(false);
      }),
      subscribe('comment', (d: any) => {
        if (d.room_id === rid && d.id) {
          setComments(prev => {
            if (prev.some(c => c.id === d.id)) return prev;
            const cmt: Comment = { id: d.id, room_id: d.room_id, user_id: d.user_id, content: d.content, created_at: new Date().toISOString(), username: d.username };
            return [...prev, cmt];
          });
          scrollCommentsToBottom();
        }
      }),
    ];
    return () => unsubs.forEach(fn => fn());
  }, [subscribe]);

  // Toast auto-dismiss
  useEffect(() => {
    if (!toast) return;
    const t = setTimeout(() => setToast(''), 3000);
    return () => clearTimeout(t);
  }, [toast]);

  // Place bid
  const handleBid = async () => {
    if (!data?.auction_session || !token) return;
    setMsg('');
    try {
      const key = `bid-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
      await buyerApi.placeBid(data.auction_session.id, amount, key, token);
      setMsg('出价成功！');
      loadRankingRef.current();
    } catch (err: any) {
      setMsg(err.message);
    }
  };

  // Send comment
  const handleComment = async () => {
    if (!token || !commentText.trim()) return;
    const text = commentText.trim();
    setCommentText('');
    try {
      const res = await fetch(`${API_HOST}/api/live-rooms/${rid}/comments`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ content: text }),
      });
      if (res.ok) await loadComments();
    } catch { /* ignore */ }
  };

  // Fetch showcase products
  const loadProducts = async () => {
    try {
      const res = await fetch(`${API_HOST}/api/live-rooms/${rid}/products`);
      const json = await res.json();
      if (json.code === 0) {
        const list: any[] = (json.data.list || []);
        // Keep only the newest (highest auction_id) per product (dedup)
        const seen = new Map<number, any>();
        for (const p of list) {
          const existing = seen.get(p.product_id);
          if (!existing || p.auction_id > existing.auction_id) {
            seen.set(p.product_id, p);
          }
        }
        setProducts(Array.from(seen.values()));
      }
    } catch { /* ignore */ }
  };

  const toggleCart = () => {
    if (!showCart) { loadProducts(); setShowCart(true); }
    else { closeCart(); }
  };
  const closeCart = () => {
    setCartClosing(true);
    setTimeout(() => { setShowCart(false); setCartClosing(false); }, 300);
  };

  // Seller: start auction for a product
  const handleStartAuction = async (product: any) => {
    if (!token) return;
    const mins = product.duration_min || 5;
    try {
      // Create new auction session
      const res = await fetch(`${API_HOST}/api/seller/auction-sessions`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          room_id: rid,
          product_id: product.product_id,
          start_price: product.start_price || '0',
          bid_increment: product.bid_increment || '10',
          duration_min: Number(mins),
          ceiling_price: product.ceiling_price || '0',
          delay_seconds: product.delay_seconds || 30,
        }),
      });
      const json = await res.json();
      if (json.code !== 0) { alert(json.message); return; }
      // Immediately start the auction
      await fetch(`${API_HOST}/api/seller/auction-sessions/${json.data.id}/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      });
      await loadProducts();
      await loadAuction();
      setEndedProduct(null);    // new auction → clear old ended card
      setRanking([]);           // new auction → clear old ranking
      setMyBid(null);
    } catch (err: any) { alert(err.message); }
  };

  // Seller: open edit modal
  const openEdit = (p: any) => {
    setEditProduct(p);
    setEditForm({
      start_price: fmt(p.start_price || '0'),
      bid_increment: fmt(p.bid_increment || '10'),
      ceiling_price: fmt(p.ceiling_price || '0'),
      duration_min: String(p.duration_min || 5),
      delay_seconds: String(p.delay_seconds || 30),
    });
  };

  // Seller: save edit
  const handleSaveEdit = async () => {
    if (!token || !editProduct?.product_id) return;
    const body: any = {};
    if (editForm.start_price) body.start_price = editForm.start_price;
    if (editForm.bid_increment) body.bid_increment = editForm.bid_increment;
    if (editForm.ceiling_price) body.ceiling_price = editForm.ceiling_price;
    if (editForm.duration_min) body.duration_min = Number(editForm.duration_min);
    if (editForm.delay_seconds) body.delay_seconds = Number(editForm.delay_seconds);
    // Update product
    const res = await fetch(`/api/seller/products/${editProduct.product_id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
    });
    const json = await res.json();
    if (json.code !== 0) { setEditToast(json.message); return; }
    // Also update auction session if exists
    if (editProduct.auction_id) {
      const sessionBody: any = {};
      if (editForm.start_price) sessionBody.start_price = editForm.start_price;
      if (editForm.bid_increment) sessionBody.bid_increment = editForm.bid_increment;
      if (editForm.ceiling_price) sessionBody.ceiling_price = editForm.ceiling_price;
      if (editForm.duration_min) sessionBody.duration_min = Number(editForm.duration_min);
      if (editForm.delay_seconds) sessionBody.delay_seconds = Number(editForm.delay_seconds);
      const sres = await fetch(`/api/seller/auction-sessions/${editProduct.auction_id}`, { method:'PUT', headers:{'Content-Type':'application/json', Authorization:`Bearer ${token}`}, body:JSON.stringify(sessionBody) });
      const sjson = await sres.json();
      if (sjson.code !== 0) { setEditToast(sjson.message); return; }
    }
    setEditProduct(null);
    loadProducts();
  };

  // Seller: cancel auction
  const handleCancelAuction = async (auctionId: number) => {
    if (!token) return;
    if (!confirm('确定取消该竞拍？')) return;
    try {
      await fetch(`${API_HOST}/api/seller/auction-sessions/${auctionId}/cancel`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ reason: '主播取消' }),
      });
      loadProducts();
      loadAuction();
    } catch (err: any) { alert(err.message); }
  };

  const session = data?.auction_session;
  const isActive = session?.status === 1;
  const isOwnRoom = user && data?.live_room?.seller_id === user.id;
  const [likeCount, setLikeCount] = useState(0);
  const [liking, setLiking] = useState(false);
  const [hearts, setHearts] = useState<{ id: number; x: number; y: number }[]>([]);
  const heartIdRef = useRef(0);
  const [showDetail, setShowDetail] = useState(false);
  const [detailProduct, setDetailProduct] = useState<any>(null); // specific product for showcase detail
  const [liveEnded, setLiveEnded] = useState(false);
  const [showEndLiveModal, setShowEndLiveModal] = useState(false);
  const [endingLive, setEndingLive] = useState(false);

  const handleEndLive = async () => {
    if (!token) return;
    setEndingLive(true);
    try {
      await fetch(`/api/seller/live-rooms/${rid}/end`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      });
      navigate('/');
    } catch (err: any) { alert(err.message); }
    setEndingLive(false);
    setShowEndLiveModal(false);
  };
  const [showEndedModal, setShowEndedModal] = useState(false);
  const [endedModalData, setEndedModalData] = useState<any>(null);
  const [paying, setPaying] = useState(false);
  const [confetti, setConfetti] = useState<{id:number;x:number;d:number;c:string}[]>([]);
  const confettiIdRef = useRef(0);
  const [fadeOut, setFadeOut] = useState(false);
  const [expireTick, setExpireTick] = useState(0);

  // Tick the expire countdown every second
  useEffect(() => {
    if (!showEndedModal || !endedModalData?.expireSec) return;
    const t = setInterval(() => setExpireTick(x => x + 1), 1000);
    return () => clearInterval(t);
  }, [showEndedModal, endedModalData?.expireSec]);

  // Auto-close non-winner modal after 5s
  useEffect(() => {
    if (!showEndedModal || !endedModalData) return;
    if (endedModalData.winnerId === user?.id) return; // winner keeps modal open
    setFadeOut(false);
    const t = setTimeout(() => setFadeOut(true), 4000);
    const t2 = setTimeout(() => { setShowEndedModal(false); setFadeOut(false); }, 5000);
    return () => { clearTimeout(t); clearTimeout(t2); };
  }, [showEndedModal, endedModalData?.winnerId, user?.id]);
  const [centerMsgs, setCenterMsgs] = useState<{id: number; text: string; type: string}[]>([]);
  const msgIdRef = useRef(0);

  const showCenterMsg = (text: string, type: string) => {
    const id = ++msgIdRef.current;
    setCenterMsgs(prev => {
      // Remove old messages of same type to prevent duplicates
      const filtered = prev.filter(m => m.type !== type);
      return [...filtered, {id, text, type}];
    });
    setTimeout(() => {
      setCenterMsgs(prev => prev.filter(m => m.id !== id));
    }, 2000);
  };
  const [showAddProduct, setShowAddProduct] = useState(false);
  const [allSellerProducts, setAllSellerProducts] = useState<any[]>([]);
  const [sessionMinId, setSessionMinId] = useState(0);
  const [editProduct, setEditProduct] = useState<any>(null);
  const [editForm, setEditForm] = useState({ start_price: '', bid_increment: '', ceiling_price: '', duration_min: '', delay_seconds: '' });
  const [editToast, setEditToast] = useState('');

  useEffect(() => { if (!editToast) return; const t = setTimeout(() => setEditToast(''), 2500); return () => clearTimeout(t); }, [editToast]);
  const [endedCountdown, setEndedCountdown] = useState(0);
  const endedTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const [endedProduct, setEndedProduct] = useState<any>(null);

  const spawnConfetti = () => {
    const colors = ['#fbbf24','#f87171','#60a5fa','#34d399','#f472b6','#a78bfa','#fb923c'];
    const batch: any[] = [];
    for (let i=0;i<40;i++) {
      batch.push({id:++confettiIdRef.current, x:Math.random()*100, d:Math.random()*3+1, c:colors[i%colors.length]});
    }
    setConfetti(batch);
    setTimeout(()=>setConfetti([]), 5000);
  };

  const showEndedProduct = (product: any) => {
    setEndedProduct(product);
  };

  // Open ended modal for winner/non-winner
  const openEndedModal = (winnerId?: number, winnerName?: string, winnerAvatar?: string, finalPrice?: string, orderId?: number, expireSec?: number) => {
    setEndedModalData({winnerId, winnerName, winnerAvatar, finalPrice, orderId, expireSec});
    setShowEndedModal(true);
    spawnConfetti();
  };

  const [showPaidModal, setShowPaidModal] = useState(false);

  // Pay from ended modal
  const handlePayFromModal = async () => {
    if (!token || !endedModalData?.orderId) return;
    setPaying(true);
    try {
      await fetch(`/api/orders/${endedModalData.orderId}/pay`, {method:'POST',headers:{'Content-Type':'application/json',Authorization:`Bearer ${token}`}});
      setPaying(false);
      setShowEndedModal(false);
      setShowPaidModal(true);
    } catch (err: any) { setPaying(false); setEditToast(err.message); }
  };

  // Auto-return 5s after auction ends
  const startEndedCountdown = () => {
    setEndedCountdown(5);
    if (endedTimerRef.current) clearInterval(endedTimerRef.current);
    endedTimerRef.current = setInterval(() => {
      setEndedCountdown(prev => {
        if (prev <= 1) {
          if (endedTimerRef.current) clearInterval(endedTimerRef.current);
          const el = document.getElementById('detail-drawer');
          if (el) { el.style.transform = 'translateY(100%)'; }
          setTimeout(() => { setShowDetail(false); setDetailProduct(null); }, 350);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);
  };
  useEffect(() => { return () => { if (endedTimerRef.current) clearInterval(endedTimerRef.current); }; }, []);
  const nextProduct = products.length > 0 ? products[0] : null;
  // Card shows: current session auction > ended > first pending
  const auctionProduct = (session && data?.product) ? data?.product : (endedProduct || nextProduct);
  const drawerProduct = detailProduct || auctionProduct;
  // drawerStatus: use detailProduct's status (from auction session), or fall back to session.status
  const drawerStatus = detailProduct ? detailProduct.status : (session ? session.status : 0);
  const cardStatus = (session && isActive) ? 1 : endedProduct ? endedProduct.status : (session ? 0 : 0);



  // Load initial like count
  useEffect(() => {
    fetch(`/api/live-rooms/${rid}/likes`)
      .then(r => r.json())
      .then(d => { if (d.code === 0) setLikeCount(d.data?.total || 0); })
      .catch(() => {});
  }, [rid]);

  const addHeart = (clientX?: number, clientY?: number) => {
    const id = ++heartIdRef.current;
    const x = clientX ? clientX - 20 : Math.random() * 100 + 100;
    const y = clientY ? clientY - 20 : 200;
    setHearts(prev => [...prev, { id, x, y }]);
    setTimeout(() => setHearts(prev => prev.filter(h => h.id !== id)), 1200);
  };

  const handleLike = async (e?: { clientX?: number; clientY?: number }) => {
    if (liking || !token) return;
    setLiking(true);
    setLikeCount(c => c + 1);
    addHeart(e?.clientX, e?.clientY);
    try {
      await fetch(`/api/live-rooms/${rid}/like`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      });
    } catch { /* ignore */ }
    setTimeout(() => setLiking(false), 500);
  };

  const statusLabel = (s: number) => ['待开始','正在拍卖','已成交','已流拍','已取消'][s] || '未知';
  const statusColor = (s: number) => ['#a0aec0','#68d391','#fbd38d','#fc8181','#718096'][s] || '#a0aec0';
  const statusBg = (s: number) => ['#2d3748','#276749','#744210','#9b2c2c','#4a5568'][s] || '#2d3748';

  // Load seller's products for "add to showcase"
  const loadAllSellerProducts = async () => {
    if (!token) return;
    try {
      const data = await fetch('/api/seller/products?page=1&page_size=100', {
        headers: { Authorization: `Bearer ${token}` },
      });
      const json = await data.json();
      setAllSellerProducts((json.data?.list || []).filter((p:any)=>p.status!==5));
    } catch {}
  };

  // Add product to showcase
  const handleAddToShowcase = async (product: any) => {
    if (!token) return;
    try {
      const res = await fetch('/api/seller/auction-sessions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({
          room_id: rid,
          product_id: product.id,
          start_price: product.start_price || '0',
          bid_increment: product.bid_increment || '10',
          ceiling_price: product.ceiling_price || '0',
          delay_seconds: product.delay_seconds || 30,
          duration_min: product.duration_min || 5,
          sort_order: products.length, // add at end
        }),
      });
      const json = await res.json();
      if (json.code !== 0) { alert(json.message); return; }
      loadProducts();
      setShowAddProduct(false);
    } catch (err: any) { alert(err.message); }
  };

  // Sort products: pending(0)/active(1) first, ended(2,3,4) at bottom
  const sortedProducts = [...products].sort((a: any, b: any) => {
    const order = (s: number) => s <= 1 ? 0 : 1;
    const oa = order(a.status), ob = order(b.status);
    if (oa !== ob) return oa - ob;
    return (a.sort_order ?? 0) - (b.sort_order ?? 0);
  });

  return (
    <div style={{ width: '100vw', height: '100vh', display: 'flex', background: '#1a1a2e', overflow: 'hidden' }}>
      {/* Toast */}
      {toast && (
        <div style={{
          position: 'fixed', top: 10, left: '50%', transform: 'translateX(-50%)', zIndex: 999,
          background: '#1a202c', color: '#fff', padding: '10px 20px', borderRadius: 20, fontSize: 14,
        }}>{toast}</div>
      )}

      {/* Edit validation toast — centered, auto-fade */}
      {editToast && (
        <div style={{
          position: 'fixed', top: '50%', left: '50%', transform: 'translate(-50%,-50%)', zIndex: 1000,
          background: 'rgba(15,15,40,0.95)', backdropFilter: 'blur(16px)',
          color: '#fff', padding: '14px 28px', borderRadius: 14,
          fontSize: 14, fontWeight: 600, textAlign: 'center',
          animation: 'toastIn 0.3s ease-out',
          pointerEvents: 'none',
        }}>{editToast}</div>
      )}

      {/* ==================== LEFT PANEL: Full Video ==================== */}
      <div style={{ width: '50%', display: 'flex', borderRight: '1px solid #2d2d4a', position: 'relative' }}>
        {/* Video Area (fills entire left panel) */}
        <div
          onDoubleClick={(e) => handleLike(e as any)}
          style={{
            flex: 1, background: '#0f0f23', display: 'flex', alignItems: 'center',
            justifyContent: 'center', color: '#fff', position: 'relative', minHeight: 0,
          }}
        >
          {/* ===== TOP-LEFT: Pill: ‹ | avatar | name / 本场点赞 N ===== */}
          <div style={{
            position: 'absolute', top: 12, left: 12, zIndex: 10,
            display: 'flex', alignItems: 'center', gap: 6,
          }}>
            {isOwnRoom && data?.live_room?.status === 1 && (
              <button onClick={() => setShowEndLiveModal(true)} style={{
                background: 'rgba(229,62,62,0.85)', color: '#fff', border: 'none',
                borderRadius: 20, padding: '6px 14px', fontSize: 13, cursor: 'pointer',
              }}>⏹ 结束</button>
            )}
            <button onClick={() => navigate('/')} style={{
              background: 'rgba(0,0,0,0.4)', color: '#fff', border: 'none',
              borderRadius: '50%', width: 34, height: 34, fontSize: 20, cursor: 'pointer',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
            }}>‹</button>
            <div style={{
              display: 'flex', alignItems: 'center', gap: 8,
              background: 'rgba(0,0,0,0.4)', borderRadius: 24,
              padding: '4px 14px 4px 4px',
            }}>
              {data?.live_room?.seller_avatar && data.live_room.seller_avatar !== '' ? (
                <img src={data.live_room.seller_avatar} alt=""
                  style={{ width: 32, height: 32, borderRadius: '50%', flexShrink: 0, objectFit: 'cover' }}
                />
              ) : (
                <div style={{
                  width: 32, height: 32, borderRadius: '50%', flexShrink: 0,
                  background: 'linear-gradient(135deg, #667eea, #764ba2)',
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  color: '#fff', fontSize: 14, fontWeight: 700,
                }}>{data?.live_room?.seller_nickname?.[0] || '主'}</div>
              )}
              <div>
                <div style={{ fontSize: 13, fontWeight: 'bold', lineHeight: 1.2 }}>
                  {data?.live_room?.seller_nickname || '主播'}
                </div>
                <div style={{ fontSize: 11, color: '#fbd38d', lineHeight: 1.2 }}>
                  本场点赞 {likeCount}
                </div>
              </div>
            </div>
          </div>

          {/* ===== TOP-RIGHT: Viewer Avatars + Count ===== */}
          <div style={{
            position: 'absolute', top: 12, right: 12, zIndex: 10,
            display: 'flex', alignItems: 'center', gap: 0,
          }}>
            {viewers.length > 0 ? (
              <>
                {viewers.map((u, i) => (
                  <div key={u.user_id} title={u.nickname} style={{
                    width: 28, height: 28, borderRadius: '50%',
                    border: '2px solid rgba(0,0,0,0.5)',
                    marginLeft: i > 0 ? -8 : 0, zIndex: viewers.length - i,
                    overflow: 'hidden', flexShrink: 0,
                  }}>
                    {u.avatar ? (
                      <img src={u.avatar} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
                    ) : (
                      <div style={{
                        width: '100%', height: '100%',
                        background: 'linear-gradient(135deg, #667eea, #764ba2)',
                        display: 'flex', alignItems: 'center', justifyContent: 'center',
                        color: '#fff', fontSize: 12, fontWeight: 700,
                      }}>{u.nickname[0]}</div>
                    )}
                  </div>
                ))}
                {onlineCount > viewers.length && (
                  <span style={{
                    marginLeft: -8, zIndex: 0,
                    background: 'rgba(0,0,0,0.55)', color: 'rgba(255,255,255,0.7)',
                    borderRadius: 20, padding: '2px 8px 2px 12px',
                    fontSize: 11, fontWeight: 600, whiteSpace: 'nowrap',
                  }}>+{onlineCount - viewers.length}</span>
                )}
              </>
            ) : (
              <span style={{
                background: 'rgba(0,0,0,0.4)', borderRadius: 20,
                padding: '4px 10px', fontSize: 13,
                display: 'flex', alignItems: 'center', gap: 4,
              }}>
                <span style={{ color: connected ? '#68d391' : '#fc8181', fontSize: 10 }}>●</span>
                {onlineCount || data?.live_room?.online_count || 0} 人
              </span>
            )}
          </div>

          {/* ===== BOTTOM-RIGHT: Like + Cart ===== */}
          <div style={{
            position: 'absolute', bottom: 16, right: 16, zIndex: 10,
            display: 'flex', alignItems: 'center', gap: 8,
          }}>
            <button onClick={(e) => handleLike({ clientX: e.clientX, clientY: e.clientY })} style={{
              background: 'rgba(0,0,0,0.4)', color: '#fff', border: 'none',
              borderRadius: 20, width: 40, height: 40, fontSize: 20, cursor: 'pointer',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              transition: 'transform 0.15s',
              transform: liking ? 'scale(1.3)' : 'scale(1)',
            }}>❤️</button>
            <button onClick={toggleCart} style={{
              background: 'rgba(0,0,0,0.4)', color: '#fff', border: 'none',
              borderRadius: 20, padding: '8px 14px', fontSize: 14, cursor: 'pointer',
              display: 'flex', alignItems: 'center', gap: 4,
            }}>
              🛒 {products.length > 0 && <span style={{ fontSize: 11, background: '#e53e3e', borderRadius: 10, padding: '1px 6px' }}>{products.length}</span>}
            </button>
          </div>

          {/* ===== Product Card (above like/cart, shows when auction exists) ===== */}
          {auctionProduct && (
            <div style={{
              position: 'absolute', bottom: 70, right: 16, zIndex: 10,
              width: 180, background: '#fff', borderRadius: 12, overflow: 'hidden',
              boxShadow: '0 2px 12px rgba(0,0,0,0.3)',
            }}>
              {/* Status badge — top-left */}
              <div style={{
                position: 'absolute', top: 10, left: 10, zIndex: 2,
                background: '#e53e3e', color: '#fff', fontSize: 10,
                fontWeight: 'bold', padding: '2px 8px', borderRadius: 10,
              }}>
                {cardStatus === 1 ? '正在拍卖' : cardStatus === 2 ? '已成交' : cardStatus === 3 ? '已流拍' : cardStatus === 4 ? '已取消' : '即将开拍'}
              </div>
              <div style={{
                height: 120, margin: 6, borderRadius: 8, position: 'relative',
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                overflow: 'hidden', background: '#e2e8f0',
              }}>
                {auctionProduct.cover_image ? (
                  <img src={auctionProduct.cover_image} alt="" style={{
                    position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover',
                  }} onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }} />
                ) : null}
                {!auctionProduct.cover_image && <span style={{ fontSize: 36, color: '#a0aec0', zIndex: 1 }}>📦</span>}
              </div>
              <div style={{ padding: '0 12px 4px', textAlign: 'center' }}>
                <div style={{ fontSize: 16, fontWeight: 'bold', color: '#e53e3e' }}>
                  ¥{fmt(auctionProduct.start_price || '0')}
                </div>
                <div style={{ fontSize: 11, color: '#a0aec0' }}>起拍价</div>
                {isOwnRoom ? (
                  (() => {
                    const activeProd = products.find((p: any) => p.status === 1);
                    if (activeProd && activeProd.auction_id) {
                      return (<>
                        <button onClick={async () => {
                          if (!confirm('确定取消当前竞拍？')) return;
                          try {
                            await fetch(`/api/seller/auction-sessions/${activeProd.auction_id}/cancel`, {
                              method: 'POST',
                              headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
                              body: JSON.stringify({ reason: '主播取消' }),
                            });
                            loadProducts();
                            loadAuction();
                          } catch (err: any) { alert(err.message); }
                        }} style={{
                          margin: '6px 0 4px', padding: '6px 24px',
                          background: '#a0aec0', color: '#fff', border: 'none',
                          borderRadius: 20, fontSize: 13, fontWeight: 'bold', cursor: 'pointer',
                        }}>取消竞拍</button>
                        {isActive && data && <Countdown endTimestampMs={data.end_timestamp_ms} serverTimeMs={data.server_time_ms} onEnd={loadAuction} />}
                      </>);
                    }
                    const firstPending = products.find((p: any) => p.status === 0);
                    return (
                      <button onClick={() => {
                        if (firstPending) handleStartAuction(firstPending);
                        else alert('橱窗中没有待竞拍的商品');
                      }} style={{
                        margin: '6px 0 8px', padding: '6px 24px',
                        background: '#e53e3e', color: '#fff', border: 'none',
                        borderRadius: 20, fontSize: 13, fontWeight: 'bold', cursor: 'pointer',
                      }}>开始竞拍</button>
                    );
                  })()
                ) : (
                  <>
                    <button onClick={() => { setMsg(''); setShowDetail(true); }} style={{
                      margin: '6px 0 4px', padding: '6px 24px',
                      background: '#3182ce', color: '#fff', border: 'none',
                      borderRadius: 20, fontSize: 13, fontWeight: 'bold', cursor: 'pointer',
                    }}>{isActive ? '出价' : '查看详细'}</button>
                    {isActive && data && <Countdown endTimestampMs={data.end_timestamp_ms} serverTimeMs={data.server_time_ms} onEnd={loadAuction} />}
                  </>
                )}
              </div>
            </div>
          )}

          {/* ===== Bottom Drawer: Auction Detail ===== */}
          {showDetail && drawerProduct && (() => {
            const closeDrawer = () => {
              const el = document.getElementById('detail-drawer');
              if (el) { el.style.transform = 'translateY(100%)'; }
              setTimeout(() => { setShowDetail(false); setDetailProduct(null); }, 350);
            };
            return (
            <>
              <div onClick={closeDrawer} style={{
                position: 'absolute', inset: 0, background: 'rgba(0,0,0,0.3)', zIndex: 30,
              }} />
              <div key={drawerProduct.product_id || drawerProduct.auction_id} id="detail-drawer" style={{
                position: 'absolute', bottom: 0, left: 0, right: 0, zIndex: 31,
                background: '#fff', borderRadius: '16px 16px 0 0', color: '#2d3748',
                padding: '0 16px 24px',
                animation: 'slideUp 0.3s ease-out',
                maxHeight: '60%', overflow: 'auto', transition: 'transform 0.35s ease-out',
              }}>
                {/* Pull-down handle */}
                <div onClick={closeDrawer} style={{
                  display:'flex', justifyContent:'center', padding:'10px 0 4px', cursor:'pointer',
                  position:'sticky', top:0, background:'#fff', zIndex:2,
                }}>
                  <svg width="28" height="16" viewBox="0 0 28 16" fill="none" style={{ opacity:0.3 }}>
                    <path d="M2 2l12 12L26 2" stroke="#6b7280" strokeWidth="3" strokeLinecap="round"/>
                  </svg>
                </div>
                {/* Countdown / Status */}
                <div style={{ textAlign: 'center', marginBottom: 16 }}>
                  {drawerStatus === 1 && data ? (
                    <>
                      <div style={{ fontSize: 12, color: '#718096', marginBottom: 4 }}>距离结束</div>
                      <Countdown endTimestampMs={data.end_timestamp_ms} serverTimeMs={data.server_time_ms} onEnd={loadAuction} />
                    </>
                  ) : drawerStatus >= 2 ? (
                    <>
                      <div style={{ fontSize: 18, fontWeight: 'bold', color: '#e53e3e', marginBottom: 8 }}>
                        {drawerStatus === 2 ? '🎉 已成交' : drawerStatus === 3 ? '📭 已流拍' : '🚫 已取消'}
                      </div>
                      <div style={{ fontSize: 14, color: '#718096' }}>拍卖已结束</div>
                      {endedCountdown > 0 && (
                        <div style={{ fontSize: 13, color: '#a0aec0', marginTop: 4 }}>
                          {endedCountdown}s 后自动返回
                        </div>
                      )}
                      <button onClick={closeDrawer} style={{
                        marginTop: 12, padding: '8px 24px', background: '#3182ce', color: '#fff',
                        border: 'none', borderRadius: 20, fontSize: 14, cursor: 'pointer',
                      }}>返回直播间</button>
                    </>
                  ) : (
                    <div style={{ fontSize: 16, fontWeight: 'bold', color: '#718096' }}>即将开拍</div>
                  )}
                </div>

                {/* Product image + name */}
                <div style={{ display: 'flex', gap: 12, alignItems: 'center', marginBottom: 12 }}>
                  <div style={{
                    width: 72, height: 72, borderRadius: 12, flexShrink: 0,
                    overflow: 'hidden', position: 'relative',
                    background: '#e2e8f0',
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                  }}>
                    {drawerProduct.cover_image ? (
                      <img src={drawerProduct.cover_image} alt="" style={{
                        position: 'absolute', inset: 0, width: '100%', height: '100%', objectFit: 'cover',
                      }} onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }} />
                    ) : null}
                    {!drawerProduct.cover_image && <span style={{ fontSize: 28, color: '#a0aec0', zIndex: 1 }}>📦</span>}
                  </div>
                  <div style={{ flex: 1 }}>
                    <div style={{ fontWeight: 'bold', fontSize: 16, marginBottom: 6 }}>
                      {drawerProduct.title || '竞拍商品'}
                    </div>
                    {drawerStatus === 1 && (
                      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginTop: 4 }}>
                        {/* Left: bid count + current price */}
                        <div style={{ fontSize: 13, color: '#718096', whiteSpace: 'nowrap' }}>
                          {data?.bid_count || session.bid_count} 次出价 · <span style={{ color: '#e53e3e', fontWeight: 'bold' }}>¥{fmt(data?.current_price || session.current_price)}</span>
                        </div>
                        {/* Highest bidder */}
                        {ranking.length > 0 && ranking[0].user_id && (
                          <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, fontSize: 12 }}>
                            <img src={ranking[0].avatar || undefined} alt="" style={{ width: 20, height: 20, borderRadius: '50%', background: '#667eea', border: '1px solid #fbbf24' }}
                              onError={(e) => { (e.target as HTMLImageElement).src = 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><circle cx="16" cy="16" r="16" fill="%23667eea"/><text x="16" y="22" text-anchor="middle" fill="white" font-size="12">'+(ranking[0].nickname?.[0]||'?')+'</text></svg>'; }} />
                            <span style={{ color: '#fbbf24', fontWeight: 600 }}>{ranking[0].nickname}</span>
                          </span>
                        )}
                        {/* Separator + user's own bid */}
                        {myBid && myBid.user_id !== (ranking[0]?.user_id || 0) && (
                          <>
                            <span style={{ color: '#d1d5db', fontSize: 16 }}>|</span>
                            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, fontSize: 12 }}>
                              <span style={{ color: '#9ca3af' }}>我</span>
                              <span style={{ color: '#e53e3e', fontWeight: 600 }}>¥{fmt(myBid.amount)}</span>
                              <span style={{ color: '#9ca3af', fontSize: 11 }}>#{myBid.rank}</span>
                            </span>
                          </>
                        )}
                      </div>
                    )}
                    <div style={{ fontSize: 13, color: '#718096', marginTop: 2 }}>
                      加价 ¥{fmt(drawerProduct.bid_increment || '10')}{drawerProduct.ceiling_price && drawerProduct.ceiling_price !== '0' ? ` · 封顶 ¥${fmt(drawerProduct.ceiling_price)}` : ''}
                    </div>
                  </div>
                </div>

                {/* Bid input (buyer only) — +/- buttons */}
                {drawerStatus === 1 && !isOwnRoom && (
                  <>
                    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 16, marginTop: 12 }}>
                      <button onClick={() => {
                        const inc = Number(drawerProduct.bid_increment || '10');
                        const cp = Number(data?.current_price || drawerProduct.current_price || '0');
                        setAmount(prev => String(Math.max(Number(prev) - inc, cp + inc)));
                      }} style={{
                        width: 40, height: 40, borderRadius: '50%', border: '2px solid #e2e8f0',
                        background: '#fff', fontSize: 24, cursor: 'pointer', display: 'flex',
                        alignItems: 'center', justifyContent: 'center', color: '#4a5568',
                      }}>−</button>
                      <div style={{ textAlign: 'center' }}>
                        {(() => {
                          const raw = Number(amount || data?.current_price || '0');
                          const ceil = Number(drawerProduct.ceiling_price || '0');
                          const display = ceil > 0 && raw > ceil ? ceil : raw;
                          return (<>
                        <div style={{ fontSize: 28, fontWeight: 'bold', color: '#e53e3e' }}>
                          ¥{fmt(String(display))}
                        </div>
                        <div style={{ fontSize: 11, color: '#a0aec0' }}>
                          {ceil > 0 && raw >= ceil
                            ? '🔨 已达封顶价'
                            : `加价幅度 ¥${fmt(drawerProduct.bid_increment || '10')}`}
                        </div>
                          </>); })()}
                      </div>
                      <button onClick={() => {
                        const inc = Number(drawerProduct.bid_increment || '10');
                        const ceil = Number(drawerProduct.ceiling_price || '0');
                        setAmount(prev => {
                          const next = Number(prev || data?.current_price || '0') + inc;
                          return String(ceil > 0 && next > ceil ? ceil : next);
                        });
                      }} style={{
                        width: 40, height: 40, borderRadius: '50%', border: '2px solid #e2e8f0',
                        background: '#fff', fontSize: 24, cursor: 'pointer', display: 'flex',
                        alignItems: 'center', justifyContent: 'center', color: '#4a5568',
                      }}>+</button>
                    </div>
                    <button onClick={handleBid} style={{
                      marginTop: 12, width: '100%', padding: '12px',
                      background: '#e53e3e', color: '#fff', border: 'none',
                      borderRadius: 24, fontSize: 16, fontWeight: 'bold', cursor: 'pointer',
                    }}>立即出价</button>
                  </>
                )}
                <div style={{ marginTop: 8, fontSize: 13, textAlign: 'center', minHeight: 20, lineHeight: '20px',
                  color: (msg || '').includes('成功') ? '#38a169' : '#e53e3e',
                  opacity: msg ? 1 : 0, transition: 'opacity 0.15s',
                }}>
                  {msg || ' '}
                </div>
              </div>
              <style>{`@keyframes slideUp { from { transform: translateY(100%); } to { transform: translateY(0); } } @keyframes slideDown { from { transform: translateY(0); } to { transform: translateY(100%); } }`}</style>
            </>
          ); })()}

          {/* Error overlay */}
          {error && (
            <div style={{ background: 'rgba(229,62,62,0.2)', color: '#fc8181', padding: '8px 20px', borderRadius: 8, fontSize: 14, zIndex: 15 }}>
              ⚠ {error}
            </div>
          )}

          {/* Floating hearts */}
          {hearts.map((h, i) => {
            const dur = 1.2 + (h.id % 2) * 0.3;
            return (
            <div key={h.id} style={{
              position: 'absolute', left: h.x, top: h.y, zIndex: 20,
              fontSize: 32, pointerEvents: 'none',
              animation: `heartFloat ${dur}s ease-out forwards`,
              filter: 'drop-shadow(0 0 5px rgba(255,100,100,0.4))',
            }}>❤️</div>
          )})}

          <style>{`
            @keyframes heartFloat {
              0%   { opacity: 0; transform: translateY(0) scale(0.3) rotate(0deg); }
              20%  { opacity: 1; transform: translateY(-20px) scale(1.3) rotate(-10deg); }
              50%  { opacity: 0.8; transform: translateY(-60px) scale(1.1) rotate(5deg); }
              100% { opacity: 0; transform: translateY(-120px) scale(0.4) rotate(15deg); }
            }
            @keyframes popIn { 0%{opacity:0;transform:scale(0.3)} 70%{transform:scale(1.1)} 100%{opacity:1;transform:scale(1)} }
            @keyframes shakeIn { 0%{opacity:0;transform:translateX(0)} 15%{opacity:1;transform:translateX(-12px)} 30%{transform:translateX(12px)} 45%{transform:translateX(-8px)} 60%{transform:translateX(8px)} 75%{transform:translateX(-4px)} 100%{transform:translateX(0)} }
            @keyframes speedLineR { 0%{opacity:0;transform:translateX(0) scaleX(0.1)} 25%{opacity:1;transform:translateX(0) scaleX(1)} 100%{opacity:0;transform:translateX(0) scaleX(1.6)} }
            @keyframes confettiFall { 0%{transform:translateY(0) rotate(0deg);opacity:1} 100%{transform:translateY(100vh) rotate(720deg);opacity:0} }
            @keyframes cmtSlideUp { from{opacity:0;transform:translateY(16px)} to{opacity:1;transform:translateY(0)} }
            @keyframes delayPulse { 0%{opacity:0;transform:scale(0.5)} 30%{opacity:1;transform:scale(1.15)} 60%{transform:scale(0.95)} 100%{opacity:1;transform:scale(1)} }
            @keyframes delayRing { 0%{opacity:0.8;transform:scale(0)} 100%{opacity:0;transform:scale(3)} }
            @keyframes toastIn { from{opacity:0;transform:translate(-50%,-50%) scale(0.9)} to{opacity:1;transform:translate(-50%,-50%) scale(1)} }
          `}</style>

          {/* Center message overlay: delay above, lead/outbid at center */}
          {centerMsgs.filter(m => m.type === 'delay').map(m => (
            <div key={m.id} style={{ position:'absolute', top:'40%', left:'50%', transform:'translate(-50%,-50%)', zIndex:25, pointerEvents:'none' }}>
              <div style={{ position:'absolute', inset:-80, overflow:'hidden', pointerEvents:'none' }}>
                {[60,120,180].map((size, i) => (
                  <div key={i} style={{ position:'absolute', top:'50%', left:'50%', width:size, height:size,
                    marginLeft:-size/2, marginTop:-size/2, borderRadius:'50%', border:'2px solid rgba(129,199,132,0.4)',
                    animation:`delayRing 1.2s ease-out ${i*0.15}s both` }} />
                ))}
              </div>
              <div style={{ fontSize:42, fontWeight:900, letterSpacing:2, color:'#81c784',
                textShadow:'0 0 40px rgba(129,199,132,0.6), 0 4px 8px rgba(0,0,0,0.5)',
                animation:'delayPulse 0.8s ease-out both' }}>{m.text}</div>
            </div>
          ))}
          {centerMsgs.filter(m => m.type !== 'delay').length > 0 && (
            (() => { const m = centerMsgs.filter(x => x.type !== 'delay').slice(-1)[0]; return (
            <div style={{ position:'absolute', inset:0, zIndex:25, display:'flex', alignItems:'center', justifyContent:'center', pointerEvents:'none' }}>
              {m.type === 'lead' && (
                <div style={{ position:'absolute', inset:0, overflow:'hidden', pointerEvents:'none' }}>
                  {[{top:'48%',w:180,delay:0},{top:'50%',w:240,delay:0.06},{top:'52%',w:150,delay:0.12},{top:'49%',w:200,delay:0.04},{top:'51%',w:130,delay:0.1}].map((l, i) => (
                    <div key={i} style={{ position:'absolute', top:l.top, left:'50%', height:3, width:l.w, borderRadius:2,
                      background:`linear-gradient(90deg, rgba(251,191,36,0.9) 0%, rgba(251,191,36,0.4) 50%, transparent 100%)`,
                      transform:'translateX(0)', animation:`speedLineR 0.8s ease-out ${l.delay}s both` }} />
                  ))}
                </div>
              )}
              <div style={{ fontSize:42, fontWeight:900, letterSpacing:2,
                color: m.type==='lead'?'#fbbf24':'#f87171',
                textShadow: m.type==='lead'?'0 0 40px rgba(251,191,36,0.6), 0 4px 8px rgba(0,0,0,0.5)':'0 0 40px rgba(248,113,113,0.6), 0 4px 8px rgba(0,0,0,0.5)',
                animation: m.type==='lead'?'popIn 0.6s cubic-bezier(0.175, 0.885, 0.32, 1.275) both':'shakeIn 0.5s ease-out both',
              }}>{m.text}</div>
            </div>
            ); })()
          )}

          {/* Video content */}
          {!error && (
            <div style={{ textAlign: 'center' }}>
              <div style={{ fontSize: 48 }}>📺</div>
              <div style={{ fontSize: 16, marginTop: 12, fontWeight: 'bold' }}>{data?.live_room?.title || '直播间'}</div>
              <div style={{ fontSize: 13, color: '#a0aec0', marginTop: 4 }}>
                {connected ? '🟢 实时' : '🔴 离线'}
              </div>
            </div>
          )}
        </div>
      </div>

      {/* ==================== RIGHT PANEL: Ranking | Comments ==================== */}
      <div style={{ width: '50%', display: 'flex', flexDirection: 'row', background: '#16213e' }}>
        {/* Ranking Section (left half of right panel) */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden', borderRight: '1px solid #2d2d4a' }}>
          <div style={{
            padding: '12px 16px', fontWeight: 'bold', fontSize: 15, color: '#fff',
            borderBottom: '1px solid #2d2d4a', background: '#1a1a3e', flexShrink: 0,
          }}>
            🏆 实时排行榜
            {polling && <span style={{ fontSize: 11, color: '#a0aec0', marginLeft: 8 }}>· 实时更新</span>}
          </div>
          <div style={{ flex: 1, overflow: 'auto' }}>
            <RankingList ranking={ranking} myBid={myBid} myUserId={user?.id} />
          </div>
        </div>

        {/* Comments Section (right half of right panel) */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          <div style={{
            padding: '10px 16px', fontWeight: 'bold', fontSize: 15, color: '#fff',
            borderBottom: '1px solid #2d2d4a', background: '#1a1a3e', flexShrink: 0,
          }}>
            💬 互动评论
          </div>

          {/* Comment List — bottom-aligned, grows upward */}
          <div ref={commentListRef} style={{
            flex: 1, overflow: 'auto', padding: '8px 12px',
            display: 'flex', flexDirection: 'column',
          }}>
            {/* Spacer pushes content to bottom when fewer items than height */}
            <div style={{ marginTop: 'auto' }} />
            {comments.length === 0 ? (
              <div style={{ color: '#718096', textAlign: 'center', padding: 30, fontSize: 13 }}>暂无评论，来发第一条吧！</div>
            ) : (
              comments.map((c, i) => (
                <div key={c.id || i} style={{
                  padding: '6px 0', borderBottom: '1px solid #1e1e3a',
                  fontSize: 13, lineHeight: 1.5,
                  animation: 'cmtSlideUp 0.35s ease-out',
                }}>
                  {c.user_id === data?.live_room?.seller_id && (
                    <span style={{
                      background: '#e53e3e', color: '#fff', fontSize: 10,
                      padding: '1px 5px', borderRadius: 4, marginRight: 4,
                      fontWeight: 'bold',
                    }}>主播</span>
                  )}
                  <span style={{ color: '#68d391', fontWeight: 'bold', marginRight: 8 }}>
                    {c.username || `用户${c.user_id}`}
                  </span>
                  <span style={{ color: '#e2e8f0' }}>{c.content}</span>
                </div>
              ))
            )}
          </div>

          {/* Comment Input */}
          <div style={{
            padding: '10px 12px', borderTop: '1px solid #2d2d4a',
            display: 'flex', gap: 8, flexShrink: 0,
          }}>
            <input
              type="text"
              value={commentText}
              onChange={e => setCommentText(e.target.value)}
              onKeyDown={e => { if (e.key === 'Enter') handleComment(); }}
              placeholder="说点什么..."
              maxLength={500}
              style={{
                flex: 1, padding: '8px 12px', border: '1px solid #2d2d4a',
                borderRadius: 20, fontSize: 13, background: '#0f0f23', color: '#fff',
              }}
            />
            <button onClick={handleComment} style={{
              padding: '8px 18px', background: '#3182ce', color: '#fff', border: 'none',
              borderRadius: 20, fontSize: 13, cursor: 'pointer', fontWeight: 'bold',
            }}>发送</button>
          </div>
        </div>
      </div>

      {/* ==================== Cart / Showcase Slide Panel ==================== */}
      {showCart && (
        <>
          <style>{`@keyframes showcaseIn { from { opacity: 0; transform: translateX(100%); } to { opacity: 1; transform: translateX(0); } } @keyframes showcaseOut { from { opacity: 1; transform: translateX(0); } to { opacity: 0; transform: translateX(100%); } }`}</style>
          <div onClick={closeCart} style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.3)', zIndex: 200 }} />
          <div style={{
            position: 'fixed', top: 0, right: 0, width: 380, height: '100vh',
            background: '#1a1a2e', zIndex: 201, overflow: 'auto',
            boxShadow: '-2px 0 16px rgba(0,0,0,0.4)', color: '#fff',
            animation: cartClosing ? 'showcaseOut 0.3s ease-in forwards' : 'showcaseIn 0.35s ease-out',
          }}>
            <div style={{
              padding: '16px', borderBottom: '1px solid #2d2d4a',
              display: 'flex', justifyContent: 'space-between', alignItems: 'center',
              position: 'sticky', top: 0, background: '#1a1a2e', zIndex: 1,
            }}>
              <h3 style={{ margin: 0, fontSize: 16 }}>🛒 {isSeller ? '管理橱窗' : '主播橱窗'} ({sortedProducts.length})</h3>
              <div style={{ display: 'flex', gap: 8 }}>
                {isSeller && isOwnRoom && data?.live_room?.status === 1 && (
                  <button onClick={() => { loadAllSellerProducts(); setShowAddProduct(true); }} style={{
                    padding: '4px 10px', background: '#3182ce', color: '#fff', border: 'none',
                    borderRadius: 6, fontSize: 12, cursor: 'pointer', whiteSpace: 'nowrap',
                  }}>+ 添加</button>
                )}
                <button onClick={closeCart} style={{ background: 'none', border: 'none', fontSize: 20, color: '#fff', cursor: 'pointer' }}>✕</button>
              </div>
            </div>

            {/* Add product sub-panel */}
            {showAddProduct && (
              <div style={{ padding: '0 16px 12px', borderBottom: '1px solid #2d2d4a' }}>
                <div style={{ fontSize: 13, fontWeight: 'bold', marginBottom: 8, color: '#a0aec0' }}>选择要添加的商品：</div>
                <div style={{ maxHeight: 200, overflow: 'auto' }}>
                  {allSellerProducts.filter((ap: any) => !products.some((p: any) => p.product_id === ap.id)).length === 0 ? (
                    <div style={{ color: '#718096', fontSize: 12, padding: 10 }}>所有商品已添加</div>
                  ) : (
                    allSellerProducts.filter((ap: any) => !products.some((p: any) => p.product_id === ap.id)).map((ap: any) => (
                      <div key={ap.id} onClick={() => handleAddToShowcase(ap)} style={{
                        display: 'flex', alignItems: 'center', gap: 8, padding: '6px 8px',
                        cursor: 'pointer', borderRadius: 6, marginBottom: 2,
                        background: '#2d2d4a',
                      }}>
                        <span style={{ fontSize: 13, flex: 1 }}>{ap.title}</span>
                        <span style={{ fontSize: 11, color: '#a0aec0' }}>起拍 ¥{fmt(ap.start_price || '0')}</span>
                        <span style={{ color: '#68d391', fontSize: 16 }}>+</span>
                      </div>
                    ))
                  )}
                </div>
                <button onClick={() => setShowAddProduct(false)} style={{
                  marginTop: 8, padding: '4px 12px', background: '#4a5568', color: '#fff',
                  border: 'none', borderRadius: 4, fontSize: 11, cursor: 'pointer',
                }}>取消</button>
              </div>
            )}

            <div style={{ padding: '12px 16px' }}>
              {sortedProducts.length === 0 ? (
                <div style={{ textAlign: 'center', color: '#718096', padding: 40, fontSize: 14 }}>暂无商品</div>
              ) : (
                sortedProducts.map((p: any, i: number) => (
                  <div key={i} style={{
                    padding: '10px 0', borderBottom: '1px solid #2d2d4a',
                    opacity: p.status > 1 ? 0.6 : 1,
                  }}>
                    <div style={{ display: 'flex', gap: 10 }}>
                      <div style={{
                        width: 56, height: 56, borderRadius: 10, flexShrink: 0,
                        background: '#3a3a5c', display: 'flex', alignItems: 'center',
                        justifyContent: 'center', overflow: 'hidden', position: 'relative',
                      }}>
                        {p.cover_image ? (
                          <img src={p.cover_image} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover', position: 'relative', zIndex: 1 }}
                            onError={(e) => { (e.target as HTMLImageElement).style.display = 'none'; }} />
                        ) : null}
                        {!p.cover_image && <span style={{ fontSize: 24, color: '#718096', fontWeight: 'bold' }}>?</span>}
                      </div>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 2 }}>
                          <span style={{
                            fontSize: 10, padding: '1px 6px', borderRadius: 6,
                            background: statusBg(p.status), color: statusColor(p.status),
                            fontWeight: 'bold', flexShrink: 0,
                          }}>{statusLabel(p.status)}</span>
                          <span style={{ fontWeight: 'bold', fontSize: 13,
                            overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                          }}>{p.title || '商品'}</span>
                        </div>
                        <div style={{ fontSize: 11, color: '#a0aec0' }}>
                          起拍 ¥{fmt(p.start_price || '0')} · 加价 ¥{fmt(p.bid_increment || '10')}
                          {p.ceiling_price && p.ceiling_price !== '0' ? ` · 封顶 ¥${fmt(p.ceiling_price)}` : ''}
                          {p.bid_count ? ` · ${p.bid_count}次出价` : ''}
                        </div>
                        {/* Sold: winner info */}
                        {p.status === 2 && p.winner_nickname && (
                          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginTop: 6, padding: '6px 8px', background: 'rgba(251,191,36,0.08)', borderRadius: 8 }}>
                            <img src={p.winner_avatar || undefined} alt="" style={{ width: 22, height: 22, borderRadius: '50%', background: '#667eea' }}
                              onError={(e) => { (e.target as HTMLImageElement).src = 'data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32"><circle cx="16" cy="16" r="16" fill="%23667eea"/><text x="16" y="22" text-anchor="middle" fill="white" font-size="12">'+(p.winner_nickname?.[0]||'?')+'</text></svg>'; }} />
                            <span style={{ fontSize: 12, color: '#fbbf24', fontWeight: 500 }}>{p.winner_nickname}</span>
                            <span style={{ fontSize: 11, color: '#a0aec0', marginLeft: 'auto' }}>落槌价 <span style={{ color: '#fbbf24', fontWeight: 700 }}>¥{fmt(p.final_price || '0')}</span></span>
                          </div>
                        )}
                        {/* Unsold/Pending: show starting price */}
                        {p.status !== 2 && (
                          <div style={{ marginTop: 4, fontSize: 11 }}><span style={{ color: '#a0aec0' }}>起拍价</span> <span style={{ color: '#fca5a5', fontWeight: 600 }}>¥{fmt(p.start_price || '0')}</span></div>
                        )}
                      </div>
                    </div>

                    {/* Seller controls — only in own live room */}
                    {isSeller && isOwnRoom && data?.live_room?.status === 1 && (
                      <div style={{ display: 'flex', gap: 6, marginTop: 6, paddingLeft: 66 }}>
                        {p.status === 0 && (
                          <>
                            <button onClick={() => handleStartAuction(p)} style={{
                              padding: '3px 10px', background: '#38a169', color: '#fff', border: 'none',
                              borderRadius: 4, fontSize: 11, cursor: 'pointer',
                            }}>开始竞拍</button>
                            <button onClick={() => openEdit(p)} style={{
                              padding: '3px 10px', background: '#6366f1', color: '#fff', border: 'none',
                              borderRadius: 4, fontSize: 11, cursor: 'pointer',
                            }}>修改</button>
                          </>
                        )}
                        {p.status === 1 && p.auction_id && (
                          <button onClick={() => handleCancelAuction(p.auction_id)} style={{
                            padding: '3px 10px', background: '#e53e3e', color: '#fff', border: 'none',
                            borderRadius: 4, fontSize: 11, cursor: 'pointer',
                          }}>取消竞拍</button>
                        )}
                        {p.status >= 2 && (
                          <span style={{ padding: '3px 10px', fontSize: 11, color: '#a0aec0' }}>
                            {p.status === 2 ? '已成交' : p.status === 3 ? '已流拍' : '已取消'}
                          </span>
                        )}
                      </div>
                    )}
                    {/* Buyer: 去看看 button for pending */}
                    {!isSeller && p.status === 0 && (
                      <div style={{ marginTop: 4, paddingLeft: 66 }}>
                        <button onClick={(e) => { e.stopPropagation(); setMsg(''); setDetailProduct(p); setShowDetail(true); }} style={{
                          padding:'4px 14px', background:'rgba(99,102,241,0.15)', color:'#a5b4fc',
                          border:'1px solid rgba(99,102,241,0.25)', borderRadius:14, fontSize:11, cursor:'pointer', fontWeight:500,
                          transition:'all 0.15s',
                        }}
                          onMouseEnter={el=>{el.currentTarget.style.background='rgba(99,102,241,0.25)';el.currentTarget.style.color='#c7d2fe'}}
                          onMouseLeave={el=>{el.currentTarget.style.background='rgba(99,102,241,0.15)';el.currentTarget.style.color='#a5b4fc'}}
                        >去看看</button>
                      </div>
                    )}
                    {/* Buyer: just show status for ended */}
                    {!isSeller && p.status >= 2 && (
                      <div style={{ marginTop: 4, paddingLeft: 66 }}>
                        <span style={{ fontSize: 11, color: '#a0aec0' }}>
                          {p.status === 2 ? '已成交' : p.status === 3 ? '已流拍' : '已取消'}
                        </span>
                      </div>
                    )}
                  </div>
                ))
              )}
            </div>
          </div>
        </>
      )}

      {/* ==================== Edit Auction Modal ==================== */}
      {editProduct && (
        <div style={{ position:'fixed', inset:0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex:300, backdropFilter:'blur(4px)' }}>
          <div style={{ background:'rgba(15,15,40,0.98)', backdropFilter:'blur(20px)', padding:24, borderRadius:16, width:380, border:'1px solid rgba(255,255,255,0.08)', color:'#e2e8f0', animation:'fadeIn 0.2s ease-out' }}>
            <h3 style={{ margin:'0 0 16px', fontWeight:700, fontSize:16 }}>修改竞拍规则 — {editProduct.title}</h3>
            {[
              {l:'起拍价',k:'start_price'},
              {l:'加价幅度',k:'bid_increment'},
              {l:'封顶价',k:'ceiling_price'},
              {l:'时长(分钟)',k:'duration_min'},
              {l:'延时(秒)',k:'delay_seconds'},
            ].map(f => (
              <div key={f.k} style={{ marginBottom:10 }}>
                <label style={{ fontSize:12, color:'rgba(148,163,184,0.6)', display:'block', marginBottom:4 }}>{f.l}</label>
                <input value={(editForm as any)[f.k]} onChange={e => setEditForm({...editForm, [f.k]: e.target.value})}
                  style={{ width:'100%', padding:'8px 12px', background:'rgba(255,255,255,0.04)', border:'1px solid rgba(255,255,255,0.08)', borderRadius:8, fontSize:14, color:'#e2e8f0', outline:'none', fontFamily:'inherit', boxSizing:'border-box' }} />
              </div>
            ))}
            <div style={{ display:'flex', gap:8, marginTop:16 }}>
              <button onClick={handleSaveEdit} style={{ flex:1, padding:10, background:'linear-gradient(135deg,#6366f1,#3b82f6)', color:'#fff', border:'none', borderRadius:8, cursor:'pointer', fontSize:14, fontWeight:600 }}>保存</button>
              <button onClick={() => setEditProduct(null)} style={{ flex:1, padding:10, background:'rgba(255,255,255,0.06)', color:'rgba(226,232,240,0.6)', border:'1px solid rgba(255,255,255,0.08)', borderRadius:8, cursor:'pointer', fontSize:14 }}>取消</button>
            </div>
          </div>
        </div>
      )}

      {/* ==================== Auction Ended Modal ==================== */}
      {showEndedModal && endedModalData && (
        <div style={{position:'fixed',inset:0,zIndex:500,display:'flex',alignItems:'center',justifyContent:'center',fontFamily:"'Noto Sans SC','PingFang SC',system-ui,sans-serif"}}>
          {/* Backdrop */}
          <div style={{position:'absolute',inset:0,background:'rgba(0,0,0,0.6)',backdropFilter:'blur(4px)'}} />

          {/* Confetti */}
          {confetti.map(c=>(
            <div key={c.id} style={{position:'fixed',top:-20,left:`${c.x}%`,width:8,height:14,background:c.c,borderRadius:2,opacity:0.8,zIndex:501,pointerEvents:'none',animation:`confettiFall ${c.d+1.5}s linear ${c.d*0.3}s forwards`}} />
          ))}

          {/* Card */}
          <div style={{position:'relative',zIndex:502,background:'#fff',borderRadius:20,padding:'32px 28px 24px',width:380,maxHeight:'85vh',overflow:'auto',boxShadow:'0 20px 60px rgba(0,0,0,0.3)',animation:'popIn 0.5s cubic-bezier(0.175,0.885,0.32,1.275) both',opacity:fadeOut?0:1,transition:'opacity 1s ease-out'}}>
            {/* Firework decoration at top */}

            {endedModalData.winnerId === user?.id ? (
              /* ===== WINNER VIEW ===== */
              <>
                <div style={{textAlign:'center',marginBottom:16,position:'relative'}}>
                  <div style={{fontSize:28,fontWeight:900,color:'#d97706',letterSpacing:2,textShadow:'0 2px 4px rgba(217,119,6,0.2)'}}>🎉 恭喜竞拍成功！</div>
                </div>
                <div style={{display:'flex',gap:14,alignItems:'center',marginBottom:20}}>
                  <img src={auctionProduct?.cover_image||''} alt="" style={{width:80,height:80,borderRadius:12,objectFit:'cover',background:'#f3f4f6'}}
                    onError={e=>{(e.target as HTMLImageElement).style.display='none'}} />
                  <div style={{flex:1}}>
                    <div style={{fontWeight:700,fontSize:16,color:'#1f2937',marginBottom:6}}>{auctionProduct?.title||'商品'}</div>
                    <div style={{fontSize:13,color:'#6b7280'}}>落槌价</div>
                    <div style={{fontSize:22,fontWeight:800,color:'#e53e3e'}}>¥{fmt(endedModalData.finalPrice||'0')}</div>
                  </div>
                </div>
                <button onClick={handlePayFromModal} disabled={paying}
                  style={{display:'block',width:'100%',padding:'14px',background:'linear-gradient(135deg,#e53e3e,#dc2626)',color:'#fff',border:'none',borderRadius:30,fontSize:16,fontWeight:700,cursor:paying?'default':'pointer',letterSpacing:1,marginBottom:12}}>
                  {paying?'支付中...':'确认地址并支付'}
                </button>
                {endedModalData.expireSec > 0 && (() => {
                  const remaining = Math.max(0, endedModalData.expireSec - expireTick);
                  return (
                    <div style={{textAlign:'center',fontSize:13,color:'#ef4444',fontWeight:600}}>
                      ⏱ 距购买失败还剩 {Math.floor(remaining/60)}分{remaining%60}秒
                    </div>
                  );
                })()}
              </>
            ) : (
              /* ===== NON-WINNER VIEW ===== */
              <>
                <div style={{textAlign:'center',marginBottom:16}}>
                  <div style={{fontSize:26,fontWeight:900,color:'#d97706',letterSpacing:1}}>🎊 恭喜成交！</div>
                </div>
                <div style={{textAlign:'center',marginBottom:16}}>
                  <img src={endedModalData.winnerAvatar||undefined} alt="" style={{width:56,height:56,borderRadius:'50%',background:'#667eea',border:'3px solid #fbbf24'}}
                    onError={e=>{(e.target as HTMLImageElement).src='data:image/svg+xml,<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 40"><circle cx="20" cy="20" r="20" fill="%23667eea"/><text x="20" y="26" text-anchor="middle" fill="white" font-size="16">'+(endedModalData.winnerName?.[0]||'?')+'</text></svg>'}} />
                  <div style={{fontWeight:700,fontSize:16,color:'#1f2937',marginTop:8}}>{endedModalData.winnerName||'买家'}</div>
                  <div style={{fontSize:13,color:'#6b7280',marginTop:4}}>经过激烈竞拍成功拍下</div>
                </div>
                <div style={{textAlign:'center'}}>
                  <div style={{fontSize:14,color:'#9ca3af'}}>落槌价</div>
                  <div style={{fontSize:32,fontWeight:900,color:'#e53e3e'}}>¥{fmt(endedModalData.finalPrice||'0')}</div>
                  <div style={{fontSize:13,color:'#9ca3af',marginTop:2}}>最终成交价</div>
                </div>
              </>
            )}
            {/* Close button — only for winner */}
            {endedModalData.winnerId === user?.id && (
              <button onClick={()=>setShowEndedModal(false)} style={{margin:'20px auto 0',width:40,height:40,borderRadius:'50%',border:'1px solid #d1d5db',background:'transparent',color:'#6b7280',fontSize:18,cursor:'pointer',display:'flex',alignItems:'center',justifyContent:'center'}}>✕</button>
            )}
          </div>
        </div>
      )}

      {/* ==================== Live Ended Modal ==================== */}
      {/* End Live confirmation modal */}
      {showEndLiveModal && (
        <div style={{
          position: 'fixed', inset: 0, zIndex: 600,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          background: 'rgba(0,0,0,0.6)', backdropFilter: 'blur(4px)',
        }} onClick={() => setShowEndLiveModal(false)}>
          <div style={{
            background: 'rgba(15,15,40,0.98)', backdropFilter: 'blur(20px)',
            borderRadius: 16, padding: '28px 32px',
            border: '1px solid rgba(255,255,255,0.08)',
            textAlign: 'center', maxWidth: 360,
            animation: 'cardIn 0.3s ease-out',
          }} onClick={e => e.stopPropagation()}>
            <div style={{ fontSize: 18, fontWeight: 700, color: '#e2e8f0', marginBottom: 12 }}>
              确定结束直播？
            </div>
            <div style={{ fontSize: 13, color: 'rgba(148,163,184,0.5)', marginBottom: 24, lineHeight: 1.5 }}>
              结束后将结算本场竞拍并生成订单
            </div>
            <div style={{ display: 'flex', gap: 10 }}>
              <button onClick={() => setShowEndLiveModal(false)} style={{
                flex: 1, padding: '10px 0',
                background: 'rgba(255,255,255,0.06)', color: 'rgba(226,232,240,0.6)',
                border: '1px solid rgba(255,255,255,0.08)', borderRadius: 10,
                fontSize: 14, cursor: 'pointer',
              }}>取消</button>
              <button onClick={handleEndLive} style={{
                flex: 1, padding: '10px 0',
                background: 'linear-gradient(135deg, #e53e3e, #dc2626)',
                color: '#fff', border: 'none', borderRadius: 10,
                fontSize: 14, fontWeight: 600, cursor: endingLive ? 'default' : 'pointer',
                opacity: endingLive ? 0.6 : 1,
              }}>{endingLive ? '结束中...' : '结束直播'}</button>
            </div>
          </div>
        </div>
      )}

      {/* ==================== Paid Success Modal ==================== */}
      {showPaidModal && (
        <div style={{position:'fixed',inset:0,zIndex:600,display:'flex',alignItems:'center',justifyContent:'center',fontFamily:"'Noto Sans SC','PingFang SC',system-ui,sans-serif"}}>
          <div style={{position:'absolute',inset:0,background:'rgba(0,0,0,0.5)',backdropFilter:'blur(4px)'}} onClick={() => setShowPaidModal(false)} />
          <div style={{position:'relative',zIndex:601,background:'#fff',borderRadius:20,padding:'36px 40px 28px',width:340,textAlign:'center',boxShadow:'0 20px 60px rgba(0,0,0,0.3)',animation:'popIn 0.4s cubic-bezier(0.175,0.885,0.32,1.275) both'}}>
            <button onClick={() => setShowPaidModal(false)} style={{position:'absolute',top:12,right:14,background:'none',border:'none',color:'#a0aec0',fontSize:18,cursor:'pointer'}}>✕</button>
            <div style={{width:64,height:64,borderRadius:'50%',background:'#10b981',display:'flex',alignItems:'center',justifyContent:'center',margin:'0 auto 16px'}}>
              <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
                <polyline points="20 6 9 17 4 12"/>
              </svg>
            </div>
            <div style={{fontSize:20,fontWeight:700,color:'#1f2937',marginBottom:6}}>支付成功</div>
            <div style={{fontSize:13,color:'#9ca3af',marginBottom:20}}>感谢您的参与，请等待卖家发货</div>
            <button onClick={() => setShowPaidModal(false)} style={{padding:'10px 40px',background:'#10b981',color:'#fff',border:'none',borderRadius:10,fontSize:14,fontWeight:600,cursor:'pointer'}}>确定</button>
          </div>
        </div>
      )}

      {liveEnded && (
        <div style={{
          position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.7)', zIndex: 999,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}>
          <div style={{
            background: '#fff', borderRadius: 16, padding: '32px 40px',
            textAlign: 'center', maxWidth: 360, position: 'relative',
          }}>
            <button onClick={() => { setLiveEnded(false); navigate('/'); }} style={{
              position: 'absolute', top: 12, right: 12, background: 'none',
              border: 'none', fontSize: 20, cursor: 'pointer', color: '#a0aec0',
            }}>✕</button>
            <div style={{ fontSize: 20, fontWeight: 'bold', color: '#2d3748', marginBottom: 8 }}>
              直播已结束
            </div>
            <div style={{ fontSize: 14, color: '#718096', marginBottom: 24 }}>
              感谢观看，期待下次再见！
            </div>
            <button onClick={() => { setLiveEnded(false); navigate('/'); }} style={{
              width: '100%', padding: '12px', background: '#3182ce', color: '#fff',
              border: 'none', borderRadius: 8, fontSize: 16, fontWeight: 'bold', cursor: 'pointer',
            }}>确定</button>
          </div>
        </div>
      )}
    </div>
  );
}
