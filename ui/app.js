const apiBase = '/api/products'

const els = {
  q:          document.getElementById('q'),
  searchBtn:  document.getElementById('searchBtn'),
  status:     document.getElementById('status'),
  results:    document.getElementById('results'),
  detail:     document.getElementById('detail'),
  detailName: document.getElementById('detail-name'),
  detailCoords:   document.getElementById('detail-coords'),
  detailWaypoint: document.getElementById('detail-waypoint'),
  requestBtn: document.getElementById('requestBtn'),
  backBtn:    document.getElementById('backBtn'),
}

let lastResults = []

function setStatus(s) { els.status.textContent = s }

async function fetchSearch(q) {
  if (!q) return []
  try {
    const res = await fetch(`${apiBase}/search?q=${encodeURIComponent(q)}`)
    if (!res.ok) throw new Error('api-error')
    return await res.json()
  } catch (e) {
    console.warn('API search failed, loading sample data', e)
    const r = await fetch('products.json')
    return r.json()
  }
}

function cropImageUrl(cropPath) {
  if (!cropPath) return null
  // crop_path is an absolute container path like /photos/crops/xxx.jpg
  // serve it via the /photos/ route the backend exposes
  return cropPath.replace('/photos/', '/photos/')
}

function renderResults(items) {
  lastResults = items
  els.results.innerHTML = ''
  if (!items || items.length === 0) {
    setStatus('No products found')
    return
  }
  setStatus(`${items.length} result${items.length === 1 ? '' : 's'} found`)

  for (const p of items) {
    const imgUrl = cropImageUrl(p.crop_path)
    const imgHtml = imgUrl
      ? `<div class="result-img"><img src="${imgUrl}" alt="${escapeHtml(p.name)}" /></div>`
      : `<div class="result-img result-img--empty"><span>No image</span></div>`

    const li = document.createElement('li')
    li.className = 'result'
    li.innerHTML = `
      ${imgHtml}
      <h3>${escapeHtml(p.name || 'Unnamed product')}</h3>
      <p>${p.nav_x.toFixed(2)}, ${p.nav_y.toFixed(2)}</p>
      <button data-id="${p.id}">GUIDE ME →</button>
    `
    li.querySelector('button').addEventListener('click', () => showDetail(p))
    els.results.appendChild(li)
  }
}

function showDetail(p) {
  els.detailName.textContent = p.name || 'Unnamed product'
  els.detailCoords.textContent   = `${p.nav_x.toFixed(3)}, ${p.nav_y.toFixed(3)}`
  els.detailWaypoint.textContent = p.waypoint_id || '—'
  els.detail.classList.remove('hidden')
  els.results.classList.add('hidden')
  els.status.classList.add('hidden')
  els.requestBtn.disabled = false
  els.requestBtn.dataset.id = p.id
}

function backToResults() {
  els.detail.classList.add('hidden')
  els.results.classList.remove('hidden')
  els.status.classList.remove('hidden')
}

async function requestNavigation(productId) {
  try {
    setStatus('Requesting navigation...')
    const res = await fetch(`${apiBase}/${productId}/navigate`, { method: 'POST' })
    if (!res.ok) throw new Error('navigate-error')
    const data = await res.json()
    setStatus(`Robot on its way — ${data.outcome}`)
    backToResults()
  } catch (e) {
    console.warn('Navigation request failed', e)
    setStatus('Navigation unavailable')
  }
}

function escapeHtml(s) {
  return String(s || '').replace(/[&<>"]/g, c => (
    { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]
  ))
}

els.searchBtn.addEventListener('click', async () => {
  const q = els.q.value.trim()
  setStatus('Searching...')
  const items = await fetchSearch(q)
  renderResults(items)
})

els.q.addEventListener('keyup', (e) => { if (e.key === 'Enter') els.searchBtn.click() })

els.backBtn.addEventListener('click', backToResults)

els.requestBtn.addEventListener('click', () => {
  const id = els.requestBtn.dataset.id
  if (id) requestNavigation(id)
})

// Initial demo load
window.addEventListener('load', async () => {
  const sample = await fetch('products.json').then(r => r.json()).catch(() => [])
  renderResults(sample)
})