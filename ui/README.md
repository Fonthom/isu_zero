# UI for Customer Product Search

This is a minimal single-page UI to search and select products for the ISU-Zero robot system.

Usage

- Serve the `ui/` folder with a static server (recommended) and open `index.html` in your browser.

Example using Python 3 built-in server:

```bash
cd ui
python3 -m http.server 8000
# then open http://localhost:8000 in your browser
```

Notes

- The UI calls the backend API at `/api/products/search?q=...`. Run the backend to enable live search.
- If the backend is unavailable or CORS blocks the request, the UI falls back to `products.json` for a demo.
- The "Request Robot" action is intentionally disabled — robot movement is not implemented yet.
