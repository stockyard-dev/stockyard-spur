package server
import "net/http"
func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) { w.Header().Set("Content-Type", "text/html"); w.Write([]byte(dashHTML)) }
const dashHTML = `<!DOCTYPE html><html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0"><title>Spur</title><link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;700&display=swap" rel="stylesheet"><style>:root{--bg:#1a1410;--bg2:#241e18;--bg3:#2e261e;--rust:#e8753a;--leather:#a0845c;--cream:#f0e6d3;--cd:#bfb5a3;--cm:#7a7060;--gold:#d4a843;--green:#4a9e5c;--red:#c94444;--blue:#5b8dd9;--mono:'JetBrains Mono',monospace}*{margin:0;padding:0;box-sizing:border-box}body{background:var(--bg);color:var(--cream);font-family:var(--mono);line-height:1.5}.hdr{padding:1rem 1.5rem;border-bottom:1px solid var(--bg3);display:flex;justify-content:space-between;align-items:center}.hdr h1{font-size:.9rem;letter-spacing:2px}.hdr h1 span{color:var(--rust)}.main{padding:1.5rem;max-width:960px;margin:0 auto}.stats{display:grid;grid-template-columns:repeat(3,1fr);gap:.5rem;margin-bottom:1rem}.st{background:var(--bg2);border:1px solid var(--bg3);padding:.6rem;text-align:center}.st-v{font-size:1.2rem;font-weight:700}.st-l{font-size:.5rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-top:.15rem}.toolbar{display:flex;gap:.5rem;margin-bottom:1rem}.search{flex:1;padding:.4rem .6rem;background:var(--bg2);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}.search:focus{outline:none;border-color:var(--leather)}.hook{background:var(--bg2);border:1px solid var(--bg3);padding:.8rem 1rem;margin-bottom:.5rem;transition:border-color .2s}.hook:hover{border-color:var(--leather)}.hook.disabled{opacity:.5}.hook-top{display:flex;justify-content:space-between;align-items:flex-start;gap:.5rem}.hook-name{font-size:.85rem;font-weight:700}.hook-url{font-size:.6rem;color:var(--blue);margin-top:.15rem;word-break:break-all}.hook-meta{font-size:.55rem;color:var(--cm);margin-top:.3rem;display:flex;gap:.5rem;flex-wrap:wrap;align-items:center}.hook-actions{display:flex;gap:.3rem;flex-shrink:0;align-items:center}.badge{font-size:.5rem;padding:.12rem .35rem;text-transform:uppercase;letter-spacing:1px;border:1px solid var(--bg3);color:var(--cm)}.toggle{position:relative;display:inline-block;width:32px;height:18px}.toggle input{opacity:0;width:0;height:0}.sl{position:absolute;cursor:pointer;inset:0;background:var(--bg3);transition:.2s;border-radius:18px}.sl:before{content:'';position:absolute;height:14px;width:14px;left:2px;bottom:2px;background:var(--cm);transition:.2s;border-radius:50%}.toggle input:checked+.sl{background:var(--green)}.toggle input:checked+.sl:before{transform:translateX(14px);background:var(--cream)}.btn{font-size:.6rem;padding:.25rem .5rem;cursor:pointer;border:1px solid var(--bg3);background:var(--bg);color:var(--cd);transition:all .2s}.btn:hover{border-color:var(--leather);color:var(--cream)}.btn-p{background:var(--rust);border-color:var(--rust);color:#fff}.btn-sm{font-size:.55rem;padding:.2rem .4rem}.modal-bg{display:none;position:fixed;inset:0;background:rgba(0,0,0,.65);z-index:100;align-items:center;justify-content:center}.modal-bg.open{display:flex}.modal{background:var(--bg2);border:1px solid var(--bg3);padding:1.5rem;width:460px;max-width:92vw}.modal h2{font-size:.8rem;margin-bottom:1rem;color:var(--rust)}.fr{margin-bottom:.6rem}.fr label{display:block;font-size:.55rem;color:var(--cm);text-transform:uppercase;letter-spacing:1px;margin-bottom:.2rem}.fr input{width:100%;padding:.4rem .5rem;background:var(--bg);border:1px solid var(--bg3);color:var(--cream);font-family:var(--mono);font-size:.7rem}.fr input:focus{outline:none;border-color:var(--leather)}.row2{display:grid;grid-template-columns:1fr 1fr;gap:.5rem}.acts{display:flex;gap:.4rem;justify-content:flex-end;margin-top:1rem}.empty{text-align:center;padding:3rem;color:var(--cm);font-style:italic;font-size:.75rem}</style></head><body>
<div class="hdr"><h1><span>&#9670;</span> SPUR</h1><button class="btn btn-p" onclick="openForm()">+ New Webhook</button></div>
<div class="main"><div class="stats" id="stats"></div><div class="toolbar"><input class="search" id="search" placeholder="Search webhooks..." oninput="render()"></div><div id="list"></div></div>
<div class="modal-bg" id="mbg" onclick="if(event.target===this)closeModal()"><div class="modal" id="mdl"></div></div>
<script>
var A='/api',items=[],editId=null;
async function load(){var r=await fetch(A+'/hooks').then(function(r){return r.json()});items=r.hooks||[];renderStats();render();}
function renderStats(){var t=items.length,active=items.filter(function(h){return h.enabled}).length,total=items.reduce(function(s,h){return s+(h.delivery_count||0)},0);
document.getElementById('stats').innerHTML='<div class="st"><div class="st-v">'+t+'</div><div class="st-l">Webhooks</div></div><div class="st"><div class="st-v" style="color:var(--green)">'+active+'</div><div class="st-l">Active</div></div><div class="st"><div class="st-v">'+total+'</div><div class="st-l">Deliveries</div></div>';}
function render(){var q=(document.getElementById('search').value||'').toLowerCase();var f=items;
if(q)f=f.filter(function(h){return(h.name||'').toLowerCase().includes(q)||(h.event||'').toLowerCase().includes(q)||(h.target_url||'').toLowerCase().includes(q)});
if(!f.length){document.getElementById('list').innerHTML='<div class="empty">No webhooks configured.</div>';return;}
var h='';f.forEach(function(w){
h+='<div class="hook'+(w.enabled?'':' disabled')+'"><div class="hook-top"><div style="flex:1"><div class="hook-name">'+esc(w.name)+'</div>';
h+='<div class="hook-url">&#8599; '+esc(w.target_url)+'</div>';
h+='</div><div class="hook-actions">';
h+='<label class="toggle"><input type="checkbox" '+(w.enabled?'checked':'')+' onchange="tog(''+w.id+'')"><span class="sl"></span></label>';
h+='<button class="btn btn-sm" onclick="openEdit(''+w.id+'')">Edit</button>';
h+='<button class="btn btn-sm" onclick="del(''+w.id+'')" style="color:var(--red)">&#10005;</button>';
h+='</div></div><div class="hook-meta">';
if(w.event)h+='<span class="badge">'+esc(w.event)+'</span>';
if(w.secret)h+='<span>&#9679; signed</span>';
if(w.last_delivered_at)h+='<span>last: '+ft(w.last_delivered_at)+'</span>';
h+='</div></div>';});
document.getElementById('list').innerHTML=h;}
async function tog(id){var h=null;for(var j=0;j<items.length;j++){if(items[j].id===id){h=items[j];break;}}if(!h)return;
await fetch(A+'/hooks/'+id,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({enabled:h.enabled?0:1})});load();}
async function del(id){if(!confirm('Delete?'))return;await fetch(A+'/hooks/'+id,{method:'DELETE'});load();}
function formHTML(hook){var i=hook||{name:'',event:'',target_url:'',secret:''};var isEdit=!!hook;
var h='<h2>'+(isEdit?'EDIT':'NEW')+' WEBHOOK</h2>';
h+='<div class="fr"><label>Name *</label><input id="f-name" value="'+esc(i.name)+'" placeholder="e.g. Deploy Notifier"></div>';
h+='<div class="fr"><label>Target URL *</label><input id="f-url" value="'+esc(i.target_url)+'" placeholder="https://example.com/webhook"></div>';
h+='<div class="row2"><div class="fr"><label>Event</label><input id="f-event" value="'+esc(i.event)+'" placeholder="e.g. push, deploy"></div>';
h+='<div class="fr"><label>Secret</label><input id="f-secret" value="'+esc(i.secret)+'" placeholder="HMAC signing key"></div></div>';
h+='<div class="acts"><button class="btn" onclick="closeModal()">Cancel</button><button class="btn btn-p" onclick="submit()">'+(isEdit?'Save':'Create')+'</button></div>';return h;}
function openForm(){editId=null;document.getElementById('mdl').innerHTML=formHTML();document.getElementById('mbg').classList.add('open');}
function openEdit(id){var h=null;for(var j=0;j<items.length;j++){if(items[j].id===id){h=items[j];break;}}if(!h)return;editId=id;document.getElementById('mdl').innerHTML=formHTML(h);document.getElementById('mbg').classList.add('open');}
function closeModal(){document.getElementById('mbg').classList.remove('open');editId=null;}
async function submit(){var name=document.getElementById('f-name').value.trim();var url=document.getElementById('f-url').value.trim();if(!name||!url){alert('Name and URL required');return;}
var body={name:name,target_url:url,event:document.getElementById('f-event').value.trim(),secret:document.getElementById('f-secret').value.trim()};
if(editId){await fetch(A+'/hooks/'+editId,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});}
else{body.enabled=1;await fetch(A+'/hooks',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});}closeModal();load();}
function ft(t){if(!t)return'';try{return new Date(t).toLocaleDateString('en-US',{month:'short',day:'numeric'})}catch(e){return t;}}
function esc(s){if(!s)return'';var d=document.createElement('div');d.textContent=s;return d.innerHTML;}
document.addEventListener('keydown',function(e){if(e.key==='Escape')closeModal();});load();
</script></body></html>`
