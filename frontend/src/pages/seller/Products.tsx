import { useState, useEffect } from 'react';
import { useAuth } from '../../store/AuthContext';
import { seller } from '../../api';
import type { ProductWithAuction } from '../../types';

const fmt = (s?: string) => (s ? (s.endsWith('.00') ? s.slice(0, -3) : s.includes('.') ? s.replace(/0+$/, '').replace(/\.$/, '') : s) : '0');

const inputBase: React.CSSProperties = {
  display:'block', width:'100%', padding:'10px 14px',
  background:'rgba(255,255,255,0.04)', border:'1px solid rgba(255,255,255,0.08)',
  borderRadius:10, fontSize:14, color:'#e2e8f0', outline:'none',
  fontFamily:'inherit', boxSizing:'border-box',
};

const labelBase: React.CSSProperties = {
  fontSize:12, color:'rgba(148,163,184,0.6)', marginBottom:4, display:'block', letterSpacing:0.5,
};

export default function Products() {
  const { token } = useAuth();
  const [products, setProducts] = useState<ProductWithAuction[]>([]);
  const [total, setTotal] = useState(0);
  const [keyword, setKeyword] = useState('');
  const [status, setStatus] = useState('-1');
  const [page, setPage] = useState(1);
  const [showAdd, setShowAdd] = useState(false);
  const [form, setForm] = useState({ title:'', description:'', cover_image:'', start_price:'0', bid_increment:'10', ceiling_price:'0' });
  const [duration, setDuration] = useState('5');
  const [delay, setDelay] = useState('30');
  const [noCeiling, setNoCeiling] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [imagePreview, setImagePreview] = useState('');
  const [editId, setEditId] = useState<number|null>(null);
  const [showStatusMenu, setShowStatusMenu] = useState(false);
  const [prodToast, setProdToast] = useState('');

  useEffect(() => { if (!prodToast) return; const t = setTimeout(() => setProdToast(''), 2500); return () => clearTimeout(t); }, [prodToast]);

  const load = async () => {
    if (!token) return;
    const params = new URLSearchParams({ keyword, status, page: String(page), page_size: '10' });
    const data = await seller.listProducts(params.toString(), token);
    setProducts(data.list); setTotal(data.total);
  };

  useEffect(() => { load(); }, [token, page, status]);

  const handleImageUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]; if (!file || !token) return;
    setUploading(true);
    try {
      const reader = new FileReader(); reader.onload = () => setImagePreview(reader.result as string); reader.readAsDataURL(file);
      const url = await seller.uploadImage(file, token); setForm({ ...form, cover_image: url });
    } catch (err: any) { alert('图片上传失败: ' + err.message); }
    setUploading(false);
  };

  const validateCeiling = () => {
    const cp = Number(noCeiling ? '0' : form.ceiling_price || '0');
    const sp = Number(form.start_price || '0');
    if (cp > 0 && cp <= sp) { setProdToast('封顶价必须大于起拍价'); return false; }
    return true;
  };

  const handleCreate = async () => {
    if (!token || !form.title) return;
    if (!form.cover_image) { alert('请上传商品图片'); return; }
    if (!validateCeiling()) return;
    try {
      await seller.createProduct({ ...form, ceiling_price: noCeiling ? '0' : (form.ceiling_price || '0'), duration_min: Number(duration), delay_seconds: Number(delay) }, token);
      setShowAdd(false); resetForm(); load();
    } catch (err: any) { alert(err.message); }
  };

  const handleUpdate = async (id: number) => {
    if (!token) return;
    if (!validateCeiling()) return;
    try {
      await seller.updateProduct(id, { ...form, ceiling_price: noCeiling ? '0' : (form.ceiling_price || '0'), duration_min: Number(duration), delay_seconds: Number(delay) }, token);
      setEditId(null); resetForm(); load();
    } catch (err: any) { setProdToast(err.message); }
  };

  const openEdit = (p: ProductWithAuction) => {
    setEditId(p.id);
    const cp = fmt(p.ceiling_price) || '0';
    const isNoCeiling = Number(p.ceiling_price || 0) === 0;
    setForm({ title:p.title||'', description:p.description||'', cover_image:p.cover_image||'', start_price:fmt(p.start_price)||'0', bid_increment:fmt(p.bid_increment)||'10', ceiling_price: isNoCeiling ? '' : cp });
    setDuration(String((p as any).duration_min || 5));
    setDelay(String((p as any).delay_seconds || 30));
    setNoCeiling(isNoCeiling);
    setImagePreview(p.cover_image||'');
  };

  const resetForm = () => {
    setForm({ title:'', description:'', cover_image:'', start_price:'0', bid_increment:'10', ceiling_price:'0' });
    setDuration('5'); setDelay('30'); setNoCeiling(false); setImagePreview('');
  };

  const handleDelete = async (id: number) => {
    if (!token || !confirm('确定下架？')) return;
    await seller.deleteProduct(id, token); load();
  };

  const statusNames: Record<number, string> = { 0:'草稿', 1:'已上架', 3:'已售' };
  const statusColors: Record<number, string> = { 0:'#9ca3af', 1:'#60a5fa', 3:'#34d399' };

  return (
    <div style={{
      minHeight:'100vh', background:'linear-gradient(180deg,#0a0a1a 0%,#0d1b2a 50%,#0f172a 100%)',
      fontFamily:"'Noto Sans SC','PingFang SC',system-ui,sans-serif", color:'#e2e8f0',
    }}>
      {/* Validation toast */}
      {prodToast && (
        <div style={{
          position: 'fixed', top: '50%', left: '50%', transform: 'translate(-50%,-50%)', zIndex: 1000,
          background: 'rgba(15,15,40,0.95)', backdropFilter: 'blur(16px)',
          color: '#fff', padding: '14px 28px', borderRadius: 14,
          fontSize: 14, fontWeight: 600, textAlign: 'center',
          animation: 'toastIn 0.3s ease-out',
          pointerEvents: 'none',
        }}>{prodToast}</div>
      )}
      <div style={{ maxWidth:1100, margin:'0 auto', padding:'32px 20px 60px', position:'relative', zIndex:1 }}>
        {/* Header */}
        <div style={{ display:'flex', justifyContent:'space-between', alignItems:'center', marginBottom:28 }}>
          <div style={{ display:'flex', alignItems:'center', gap:12 }}>
            <div style={{ width:4, height:24, borderRadius:2, background:'#6366f1' }} />
            <h1 style={{ margin:0, fontSize:24, fontWeight:700, background:'linear-gradient(135deg,#e2e8f0,#a5b4fc)', WebkitBackgroundClip:'text', WebkitTextFillColor:'transparent' }}>商品管理 ({total})</h1>
          </div>
          <button onClick={()=>{resetForm();setShowAdd(true);}} style={{
            padding:'10px 22px', background:'linear-gradient(135deg,#6366f1,#3b82f6)', color:'#fff', border:'none', borderRadius:10, cursor:'pointer', fontSize:14, fontWeight:600,
            transition:'all 0.2s',
          }} onMouseEnter={e=>{e.currentTarget.style.transform='scale(1.04)';e.currentTarget.style.boxShadow='0 4px 16px rgba(99,102,241,0.3)'}} onMouseLeave={e=>{e.currentTarget.style.transform='scale(1)';e.currentTarget.style.boxShadow='none'}}>+ 添加商品</button>
        </div>

        {/* Search & Filter */}
        <div style={{ display:'flex', gap:10, marginBottom:24 }}>
          <input placeholder="搜索商品" value={keyword} onChange={e=>setKeyword(e.target.value)} onKeyDown={e=>e.key==='Enter'&&load()}
            style={{ flex:1, ...inputBase }} />
          <div style={{ position:'relative', width:140 }}>
            <div onClick={()=>setShowStatusMenu(!showStatusMenu)} style={{ ...inputBase, display:'flex', alignItems:'center', justifyContent:'space-between', cursor:'pointer' }}>
              <span style={{ color: status==='-1'?'rgba(148,163,184,0.4)':'#e2e8f0', fontSize:14 }}>{{'-1':'全部状态','0':'草稿','1':'已上架','3':'已售'}[status]||'全部状态'}</span>
              <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="rgba(148,163,184,0.4)" strokeWidth="2" style={{ transform:showStatusMenu?'rotate(180deg)':'rotate(0)', transition:'transform 0.2s' }}><path d="M6 9l6 6 6-6"/></svg>
            </div>
            {showStatusMenu && <>
              <div onClick={()=>setShowStatusMenu(false)} style={{ position:'fixed', inset:0, zIndex:5 }} />
              <div style={{ position:'absolute', top:48, left:0, right:0, zIndex:100, background:'rgba(15,15,40,0.98)', backdropFilter:'blur(16px)', borderRadius:10, border:'1px solid rgba(255,255,255,0.08)', overflow:'hidden', animation:'fadeIn 0.2s ease-out' }}>
                {[{v:'-1',l:'全部状态'},{v:'0',l:'草稿'},{v:'1',l:'已上架'},{v:'3',l:'已售'}].map(o=>(
                  <div key={o.v} onClick={()=>{setStatus(o.v);setPage(1);setShowStatusMenu(false);}} style={{ padding:'10px 14px', cursor:'pointer', fontSize:14, color:status===o.v?'#a78bfa':'#cbd5e0', background:status===o.v?'rgba(139,92,246,0.1)':'transparent', transition:'background 0.15s' }}
                    onMouseEnter={e=>{if(status!==o.v)e.currentTarget.style.background='rgba(255,255,255,0.04)'}} onMouseLeave={e=>{if(status!==o.v)e.currentTarget.style.background='transparent'}}>{o.l}</div>
                ))}
              </div>
            </>}
          </div>
          <button onClick={load} style={{ padding:'10px 18px', background:'rgba(255,255,255,0.06)', border:'1px solid rgba(255,255,255,0.08)', borderRadius:10, color:'rgba(226,232,240,0.6)', cursor:'pointer', fontSize:14 }}>搜索</button>
        </div>

        {/* Product list */}
        {products.map((p, i) => (
          <div key={p.id} style={{
            background:'rgba(255,255,255,0.02)', backdropFilter:'blur(12px)', borderRadius:14, padding:'14px 18px',
            marginBottom:10, display:'flex', alignItems:'center', border:'1px solid rgba(255,255,255,0.04)',
            animation:`cardIn 0.4s ease-out ${i*0.04}s both`,
            transition:'transform 0.2s',
          }} onMouseEnter={e=>e.currentTarget.style.transform='translateX(4px)'} onMouseLeave={e=>e.currentTarget.style.transform='translateX(0)'}>
            {/* Image */}
            <div style={{ width:56, height:56, borderRadius:10, flexShrink:0, marginRight:14, background:'rgba(99,102,241,0.08)', display:'flex', alignItems:'center', justifyContent:'center', overflow:'hidden', position:'relative' }}>
              {p.cover_image ? <img src={p.cover_image} alt="" style={{ width:'100%', height:'100%', objectFit:'cover' }} onError={e=>{(e.target as HTMLImageElement).style.display='none'}} /> : null}
              {!p.cover_image && <span style={{ fontSize:22, color:'rgba(148,163,184,0.3)', fontWeight:'bold' }}>?</span>}
            </div>
            {/* Info */}
            <div style={{ flex:1 }}>
              <div style={{ fontWeight:600, fontSize:14, marginBottom:4 }}>{p.title}</div>
              <div style={{ fontSize:13, color:'#cbd5e0' }}>
                起拍 ¥{fmt(p.start_price||'0')} · 加价 ¥{fmt(p.bid_increment||'10')}{Number(p.ceiling_price)>0?` · 封顶 ¥${fmt(p.ceiling_price)}`:' · 上不封顶'}
              </div>
              {p.status !== 3 && p.status !== 5 && p.current_price && <div style={{ fontSize:13, color:'#fca5a5', fontWeight:600, marginTop:2 }}>当前价: ¥{fmt(p.current_price)}</div>}
              {p.final_price && <div style={{ fontSize:13, color:'#34d399', fontWeight:600, marginTop:2 }}>成交价: ¥{fmt(p.final_price)}</div>}
              {(p as any).auction_start && <div style={{ fontSize:12, color:'#94a3b8', marginTop:2 }}>开拍: {new Date((p as any).auction_start).toLocaleString('zh-CN', {month:'numeric',day:'numeric',hour:'2-digit',minute:'2-digit'})}</div>}
            </div>
            {/* Status */}
            <span style={{ padding:'3px 10px', borderRadius:20, fontSize:11, fontWeight:600, color:statusColors[p.status]||'#a0aec0', background:`${statusColors[p.status]||'#a0aec0'}18`, marginRight:12 }}>
              {p.status_name || statusNames[p.status]}
            </span>
            {/* Actions */}
            <div style={{ display:'flex', gap:6 }}>
              {p.status === 0 && <button onClick={()=>openEdit(p)} style={btnSm('#6366f1')}>修改</button>}
              {p.status === 0 && <button onClick={async()=>{await seller.updateProduct(p.id, {status:1} as any, token!); load();}} style={btnSm('#10b981')}>上架</button>}
              {p.status === 1 && <button onClick={async()=>{await seller.updateProduct(p.id, {status:0} as any, token!); load();}} style={btnSm('#f59e0b')}>下架</button>}
              {p.status === 3 && <button onClick={()=>handleDelete(p.id)} style={btnSm('#e53e3e')}>删除</button>}
              {(p.status === 0 || p.status === 1) && <button onClick={()=>handleDelete(p.id)} style={btnSm('#e53e3e')}>删除</button>}
            </div>
          </div>
        ))}
        {products.length===0 && <div style={{ textAlign:'center', padding:50, color:'rgba(148,163,184,0.3)' }}>暂无商品</div>}

        {/* Pagination */}
        {total>10 && (
          <div style={{ textAlign:'center', marginTop:20 }}>
            <button disabled={page<=1} onClick={()=>setPage(p=>p-1)} style={{ ...btnSmBg, opacity:page<=1?0.3:1 }}>上一页</button>
            <span style={{ margin:'0 14px', fontSize:14, color:'rgba(148,163,184,0.5)' }}>{page}/{Math.ceil(total/10)}</span>
            <button disabled={page>=Math.ceil(total/10)} onClick={()=>setPage(p=>p+1)} style={{ ...btnSmBg, opacity:page>=Math.ceil(total/10)?0.3:1 }}>下一页</button>
          </div>
        )}
      </div>

      {/* Add/Edit Modal */}
      {(showAdd || editId) && (
        <div style={{ position:'fixed', inset:0, background:'rgba(0,0,0,0.6)', display:'flex', alignItems:'center', justifyContent:'center', zIndex:100, backdropFilter:'blur(4px)' }}>
          <div style={{ background:'rgba(15,15,40,0.98)', backdropFilter:'blur(20px)', padding:28, borderRadius:20, width:460, maxHeight:'90vh', overflow:'auto', border:'1px solid rgba(255,255,255,0.08)', animation:'fadeIn 0.3s ease-out' }}>
            <h2 style={{ margin:'0 0 20px', fontWeight:700, background:'linear-gradient(135deg,#e2e8f0,#a5b4fc)', WebkitBackgroundClip:'text', WebkitTextFillColor:'transparent' }}>{editId?'修改商品':'添加商品'}</h2>
            <div style={labelBase}>名称</div><input placeholder="商品名称" value={form.title} onChange={e=>setForm({...form,title:e.target.value})} style={inputBase} />
            <div style={{...labelBase, marginTop:12}}>描述</div><textarea placeholder="商品描述" value={form.description} onChange={e=>setForm({...form,description:e.target.value})} rows={3} style={{...inputBase, resize:'vertical'}} />
            <div style={{...labelBase, marginTop:12}}>图片</div>
            <label style={{
              display: 'flex', alignItems: 'center', justifyContent: 'center', gap: 8,
              padding: '10px 16px', borderRadius: 10, cursor: 'pointer',
              background: 'rgba(255,255,255,0.04)', border: '1px dashed rgba(255,255,255,0.12)',
              color: 'rgba(148,163,184,0.6)', fontSize: 13, transition: 'all 0.2s',
            }}
              onMouseEnter={e => { e.currentTarget.style.borderColor = 'rgba(99,102,241,0.4)'; e.currentTarget.style.color = '#a5b4fc'; }}
              onMouseLeave={e => { e.currentTarget.style.borderColor = 'rgba(255,255,255,0.12)'; e.currentTarget.style.color = 'rgba(148,163,184,0.6)'; }}
            >
              <span style={{ fontSize: 18 }}>📷</span> {uploading ? '上传中...' : '点击选择图片'}
              <input type="file" accept="image/*" onChange={handleImageUpload} style={{ display: 'none' }} />
            </label>
            {(imagePreview || form.cover_image) && <img src={imagePreview||form.cover_image} alt="" style={{ width:'100%', maxHeight:180, objectFit:'contain', borderRadius:8, marginTop:8, border:'1px solid rgba(255,255,255,0.06)' }} />}
            <div style={{ display:'flex', gap:10, marginTop:12 }}>
              <div style={{ flex:1 }}><div style={labelBase}>起拍价</div><input placeholder="0" value={form.start_price} onChange={e=>setForm({...form,start_price:e.target.value})} style={inputBase} /></div>
              <div style={{ flex:1 }}><div style={labelBase}>加价幅度</div><input placeholder="10" value={form.bid_increment} onChange={e=>setForm({...form,bid_increment:e.target.value})} style={inputBase} /></div>
            </div>
            <div style={{...labelBase, marginTop:12}}>封顶价</div>
            <div style={{ display:'flex', alignItems:'center', gap:10 }}>
              <div onClick={()=>setNoCeiling(!noCeiling)} style={{ cursor:'pointer', display:'flex', alignItems:'center', gap:6, flexShrink:0 }}>
                <div style={{
                  width:18, height:18, borderRadius:'50%',
                  border: noCeiling?'2px solid #818cf8':'2px solid rgba(255,255,255,0.2)',
                  background: noCeiling?'#818cf8':'transparent',
                  display:'flex', alignItems:'center', justifyContent:'center',
                  transition:'all 0.2s',
                }}>
                  {noCeiling && <span style={{ color:'#fff', fontSize:11, fontWeight:'bold' }}>✓</span>}
                </div>
                <span style={{ fontSize:12, color:noCeiling?'#cbd5e0':'rgba(148,163,184,0.5)', whiteSpace:'nowrap' }}>不设封顶价</span>
              </div>
              <input placeholder="封顶价" value={noCeiling?'':form.ceiling_price} disabled={noCeiling}
                onChange={e=>setForm({...form,ceiling_price:e.target.value})} style={{...inputBase, flex:1, opacity:noCeiling?0.3:1}} />
            </div>
            <div style={{ display:'flex', gap:10, marginTop:12 }}>
              <div style={{ flex:1 }}><div style={labelBase}>竞拍时长(分钟)</div><input placeholder="5" value={duration} onChange={e=>setDuration(e.target.value)} style={inputBase} /></div>
              <div style={{ flex:1 }}><div style={labelBase}>延时(秒)</div><input placeholder="30" value={delay} onChange={e=>setDelay(e.target.value)} style={inputBase} /></div>
            </div>
            <div style={{ display:'flex', gap:10, marginTop:20 }}>
              <button onClick={()=>editId?handleUpdate(editId):handleCreate()} style={{ flex:1, padding:12, background:'linear-gradient(135deg,#6366f1,#3b82f6)', color:'#fff', border:'none', borderRadius:10, cursor:'pointer', fontSize:15, fontWeight:600 }}>{editId?'保存修改':'创建'}</button>
              <button onClick={()=>{setShowAdd(false);setEditId(null);resetForm();}} style={{ flex:1, padding:12, background:'rgba(255,255,255,0.06)', color:'rgba(226,232,240,0.6)', border:'1px solid rgba(255,255,255,0.08)', borderRadius:10, cursor:'pointer', fontSize:15 }}>取消</button>
            </div>
          </div>
        </div>
      )}
      <style>{`@keyframes cardIn{from{opacity:0;transform:translateY(12px)}to{opacity:1;transform:translateY(0)}}@keyframes fadeIn{from{opacity:0}to{opacity:1}}@keyframes toastIn{from{opacity:0;transform:translate(-50%,-50%) scale(0.9)}to{opacity:1;transform:translate(-50%,-50%) scale(1)}}`}</style>
    </div>
  );
}

const btnSm = (bg: string): React.CSSProperties => ({
  padding:'5px 14px', background:bg, color:'#fff', border:'none', borderRadius:6, cursor:'pointer', fontSize:12, fontWeight:500,
});
const btnSmBg: React.CSSProperties = {
  padding:'8px 18px', background:'rgba(255,255,255,0.04)', color:'rgba(226,232,240,0.6)', border:'1px solid rgba(255,255,255,0.06)', borderRadius:8, cursor:'pointer', fontSize:13,
};
