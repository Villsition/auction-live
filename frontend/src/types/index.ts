// ========== Auth ==========
export interface User {
  id: number;
  username: string;
  nickname: string;
  avatar: string;
  role: number;
  status: number;
}

export interface LoginInput {
  username: string;
  password: string;
}

export interface RegisterInput {
  username: string;
  password: string;
  nickname: string;
  role?: number;
}

// ========== Auction ==========
export interface LiveRoom {
  id: number;
  seller_id: number;
  title: string;
  cover_image: string;
  stream_url: string;
  status: number; // 0=offline 1=live 2=ended
  online_count: number;
  total_likes?: number;
  seller_nickname?: string;
  seller_avatar?: string;
}

export interface AuctionSession {
  id: number;
  room_id: number;
  product_id: number;
  start_price: string;
  current_price: string;
  ceiling_price: string;
  bid_increment: string;
  delay_seconds: number;
  start_time: string | null;
  planned_end_time: string;
  actual_end_time: string | null;
  winner_id: number | null;
  final_price: string | null;
  bid_count: number;
  status: number; // 0=pending 1=active 2=sold 3=unsold 4=cancelled
  product?: Product;
}

export interface Product {
  id: number;
  title: string;
  description: string;
  cover_image: string;
  images: string[];
  start_price: string;
  ceiling_price: string;
  bid_increment: string;
  duration_min?: number;
  delay_seconds?: number;
  status: number;
  status_name?: string;
}

export interface ProductWithAuction extends Product {
  auction_id?: number;
  auction_status?: number;
  current_price?: string;
  final_price?: string;
  bid_count?: number;
}

export interface RoomAuction {
  live_room: LiveRoom;
  auction_session: AuctionSession | null;
  product: Product | null;
  current_price: string;
  bid_count: number;
  end_timestamp_ms: number;
  server_time_ms: number;
  remaining_ms: number;
  next_bid: string;
}

// ========== Ranking ==========
export interface RankItem {
  rank: number;
  user_id: number;
  nickname: string;
  avatar: string;
  amount: string;
  time: string;
}

export interface RankingResp {
  ranking: RankItem[];
  my_bid: RankItem | null;
}

// ========== Bid ==========
export interface BidResult {
  rank: number;
  ceiling_deal: boolean;
  bid: Bid;
}

export interface Bid {
  id: number;
  auction_id: number;
  user_id: number;
  amount: string;
  bid_time: string;
  nickname?: string;
}

// ========== Order ==========
export interface Order {
  id: number;
  order_no: string;
  amount: string;
  status: number; // 0=unpaid 1=paid 2=shipped 3=completed
  remaining_sec: number;
  expires_at: string;
}

export interface SellerOrderItem extends Order {
  product_title: string;
  product_image: string;
  buyer_nickname: string;
  buyer_avatar: string;
}

// ========== Notifications ==========
export interface Notification {
  id: number;
  title: string;
  content: string;
  type: number;
  is_read: number;
  related_id: number;
  created_at: string;
}

// ========== Dashboard ==========
export interface DashboardData {
  product_stats: Record<string, number>;
  auction_stats: Record<string, number>;
  order_stats: Record<string, number>;
  revenue_total: string;
  active_bidding: Array<{
    id: number;
    product_title: string;
    cover_image: string;
    current_price: string;
    bid_count: number;
    remaining_ms: number;
  }>;
}

// ========== WebSocket Events ==========
export interface WSBidEvent {
  type: 'bid' | 'delay_extend' | 'ceiling_deal';
  auction_id: number;
  amount: string;
  user_id: number;
  nickname: string;
  rank: number;
  bid_count: number;
  ceiling_deal: boolean;
  delay_extend: boolean;
  final_delay: boolean;
  new_end_time_ms: number;
}

export interface WSAuctionEvent {
  type: 'auction_start' | 'auction_end' | 'auction_cancel';
  auction_id: number;
  status: string;
  winner_id?: number;
  final_price?: string;
  message: string;
}

export interface WSOutbidEvent {
  type: 'outbid';
  auction_id: number;
  user_id: number;
  new_amount: string;
  my_rank: number;
}

export interface WSNewLeaderEvent {
  type: 'new_leader';
  auction_id: number;
  old_leader_id: number;
  new_leader_id: number;
  amount: string;
  message: string;
}

export type WSMessage = WSBidEvent | WSAuctionEvent | WSOutbidEvent | WSNewLeaderEvent;
