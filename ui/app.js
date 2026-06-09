const apiBase = '/api/products'

const els = {
  q: document.getElementById('q'),
  searchBtn: document.getElementById('searchBtn'),
  status: document.getElementById('status'),
  results: document.getElementById('results'),
  detail: document.getElementById('detail'),
  detailName: document.getElementById('detail-name'),
  detailCoords: document.getElementById('detail-coords'),
  detailWaypoint: document.getElementById('detail-waypoint'),
  requestBtn: document.getElementById('requestBtn'),
  backBtn: document.getElementById('backBtn'),
}

let lastResults = []

function setStatus(s){ els.status.textContent = s }

async function fetchSearch(q){
  if (!q) return []
  try{
    const res = await fetch(`${apiBase}/search?q=${encodeURIComponent(q)}`)
    if (!res.ok) throw new Error('api-error')
    return await res.json()
  }catch(e){
    // fallback to sample data
    console.warn('API search failed, loading sample data', e)
    const r = await fetch('products.json')
    return r.json()
  }
}

function renderResults(items){
  lastResults = items
  els.results.innerHTML = ''
  if (!items || items.length === 0){
    setStatus('No products found')
    return
  }
  setStatus(`${items.length} products`) 
  for (const p of items){
    const li = document.createElement('li')
    li.className = 'result'
    li.innerHTML = `<h3>${escapeHtml(p.name)}</h3>
      <p>Coords: ${p.nav_x.toFixed(2)}, ${p.nav_y.toFixed(2)}</p>
      <button data-id="${p.id}">Select</button>`
    li.querySelector('button').addEventListener('click', ()=> showDetail(p))
    els.results.appendChild(li)
  }
}

function showDetail(p){
  els.detailName.textContent = p.name
  els.detailCoords.textContent = `${p.nav_x.toFixed(3)}, ${p.nav_y.toFixed(3)}`
  els.detailWaypoint.textContent = p.waypoint_id || '—'
  els.detail.classList.remove('hidden')
  els.results.classList.add('hidden')
  els.status.classList.add('hidden')
  // Request button is disabled for now — UI only
  els.requestBtn.disabled = true
  els.requestBtn.title = 'Robot movement not implemented yet'
}

function backToResults(){
  els.detail.classList.add('hidden')
  els.results.classList.remove('hidden')
  els.status.classList.remove('hidden')
}

function escapeHtml(s){ return String(s||'').replace(/[&<>"]/g,c=>({ '&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c])) }

els.searchBtn.addEventListener('click', async ()=>{
  const q = els.q.value.trim()
  setStatus('Searching...')
  const items = await fetchSearch(q)
  renderResults(items)
})

els.q.addEventListener('keyup', (e)=>{ if (e.key === 'Enter') els.searchBtn.click() })
els.backBtn.addEventListener('click', backToResults)

// initial demo load: fetch sample list
window.addEventListener('load', async ()=>{
  const sample = await fetch('products.json').then(r=>r.json()).catch(()=>[])
  renderResults(sample)
})
