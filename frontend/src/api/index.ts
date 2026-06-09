import type {
  LoginInput, RegisterInput, User,
  Product, ProductWithAuction, AuctionSession,
  LiveRoom, RoomAuction, RankingResp,
  BidResult, Order, Notification, DashboardData,
} from '../types';

const BASE = '/api';

function headers(token?: string): Record<string, string> {
  const h: Record<string, string> = { 'Content-Type': 'application/json' };
  if (token) h['Authorization'] = `Bearer ${token}`;
  return h;
}

async function get<T>(url: string, token?: string): Promise<T> {
  const res = await fetch(BASE + url, { headers: headers(token) });
  const json = await res.json();
  if (json.code !== 0) throw new Error(json.message);
  return json.data;
}

async function post<T>(url: string, body: unknown, token?: string): Promise<T> {
  const res = await fetch(BASE + url, {
    method: 'POST',
    headers: headers(token),
    body: JSON.stringify(body),
  });
  const json = await res.json();
  if (json.code !== 0) throw new Error(json.message);
  return json.data;
}

// ========== Auth ==========
export const auth = {
  login: (input: LoginInput) => post<{ token: string; user: User }>('/auth/login', input),
  register: (input: LoginInput & { nickname: string; role?: number }) =>
    post<{ token: string; user: User }>('/auth/register', input),
};

// ========== Public ==========
export const publicApi = {
  serverTime: () => get<{ server_time: number }>('/server-time'),
  liveRooms: (page = 1, keyword?: string) =>
    get<LiveRoom[]>(`/live-rooms?page=${page}&page_size=20${keyword ? `&keyword=${encodeURIComponent(keyword)}` : ''}`),
  liveRoom: (id: number) => get<LiveRoom>(`/live-rooms/${id}`),
  products: (page = 1) => get<Product[]>(`/products?page=${page}&page_size=20`),
  categories: () => get<{ id: number; name: string }[]>('/categories'),
  auctionSessions: (page = 1) =>
    get<AuctionSession[]>(`/auction-sessions?page=${page}&page_size=20`),
  roomAuction: (roomId: number) => get<RoomAuction>(`/live-rooms/${roomId}/auction`),
  roomProducts: (roomId: number) => get<{ list: any[] }>(`/live-rooms/${roomId}/products`),
  bidRanking: (auctionId: number, top = 20) =>
    get<RankingResp>(`/auction-sessions/${auctionId}/ranking?top=${top}`),
};

// ========== Buyer ==========
export const buyer = {
  placeBid: (auctionId: number, amount: string, idempotencyKey: string, token: string) =>
    post<BidResult>('/bids', {
      auction_id: auctionId,
      amount,
      idempotency_key: idempotencyKey,
    }, token),
  myBids: (page = 1, token: string) =>
    get<{ list: BidResult[]; total: number }>(`/bids/mine?page=${page}&page_size=20`, token),
  myOrders: (token: string) =>
    get<Order[]>('/orders', token),
  payOrder: (orderId: number, token: string) =>
    post<Order>(`/orders/${orderId}/pay`, {}, token),
  confirmAddress: (orderId: number, address: string, token: string) =>
    post<Order>(`/orders/${orderId}/address`, { address }, token),
  notifications: (token: string) =>
    get<Notification[]>('/notifications', token),
  markRead: (id: number, token: string) =>
    post(`/notifications/${id}/read`, {}, token),
  sendComment: (roomId: number, content: string, token: string) =>
    post(`/live-rooms/${roomId}/comments`, { content }, token),
  comments: (roomId: number) =>
    get(`/live-rooms/${roomId}/comments`),
};

// ========== Seller ==========
export const seller = {
  dashboard: (token: string) => get<DashboardData>('/seller/dashboard', token),
  // Products
  createProduct: (data: Partial<Product>, token: string) =>
    post<Product>('/seller/products', data, token),
  updateProduct: async (id: number, data: Partial<Product>, token: string) => {
    const res = await fetch(BASE + `/seller/products/${id}`, { method: 'PUT', headers: headers(token), body: JSON.stringify(data) });
    const json = await res.json();
    if (json.code !== 0) throw new Error(json.message);
    return json.data;
  },
  listProducts: (params: string, token: string) =>
    get<{ list: ProductWithAuction[]; total: number }>(`/seller/products?${params}`, token),
  deleteProduct: async (id: number, token: string) => {
    const res = await fetch(BASE + `/seller/products/${id}`, { method: 'DELETE', headers: headers(token) });
    const json = await res.json();
    if (json.code !== 0) throw new Error(json.message);
  },
  // Live rooms
  createRoom: (data: Partial<LiveRoom>, token: string) =>
    post<LiveRoom>('/seller/live-rooms', data, token),
  listRooms: (token: string) =>
    get<LiveRoom[]>('/seller/live-rooms', token),
  startLive: (roomId: number, token: string) =>
    post(`/seller/live-rooms/${roomId}/start`, {}, token),
  endLive: (roomId: number, token: string) =>
    post(`/seller/live-rooms/${roomId}/end`, {}, token),
  // Auction sessions
  createAuction: (data: Record<string, unknown>, token: string) =>
    post<AuctionSession>('/seller/auction-sessions', data, token),
  listAuctions: (token: string) =>
    get<AuctionSession[]>('/seller/auction-sessions', token),
  startAuction: (id: number, token: string) =>
    post<AuctionSession>(`/seller/auction-sessions/${id}/start`, {}, token),
  cancelAuction: (id: number, reason: string, token: string) =>
    post(`/seller/auction-sessions/${id}/cancel`, { reason }, token),
  uploadVideo: async (file: File, token: string): Promise<string> => {
    const form = new FormData();
    form.append('file', file);
    const res = await fetch(BASE + '/upload/video', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      body: form,
    });
    const json = await res.json();
    if (json.code !== 0) throw new Error(json.message);
    return json.data.url;
  },
  uploadImage: async (file: File, token: string): Promise<string> => {
    const form = new FormData();
    form.append('file', file);
    const res = await fetch(BASE + '/seller/upload/image', {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` },
      body: form,
    });
    const json = await res.json();
    if (json.code !== 0) throw new Error(json.message);
    return json.data.url;
  },
  sellerOrders: (token: string) =>
    get<Order[]>('/seller/orders', token),
};
