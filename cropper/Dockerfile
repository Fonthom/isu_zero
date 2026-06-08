FROM python:3.14-slim

WORKDIR /app

# Copy pre-downloaded wheels and install offline
COPY wheels/ wheels/
COPY requirements.txt .
RUN pip install --no-cache-dir --no-index --find-links=wheels/ -r requirements.txt

# Copy pre-downloaded model weights — no internet access needed at build time
COPY yolov8n.pt .

COPY . .

CMD ["python", "main.py"]