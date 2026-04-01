FROM node:18-alpine AS builder
WORKDIR /app
COPY services/iris/client/package*.json ./client/
RUN cd client && npm ci
COPY services/iris/client/ ./client/
RUN cd client && npm run build

FROM node:18-alpine
WORKDIR /app
COPY services/iris/package*.json ./
RUN npm ci --production
COPY services/iris/server/ ./server/
COPY --from=builder /app/client/build ./client/build
HEALTHCHECK --interval=10s --timeout=3s --retries=3 \
  CMD curl -sf http://localhost:${PORT:-3001}/api/health || exit 1
ENV NODE_ENV=production
CMD ["node", "server/index.js"]
