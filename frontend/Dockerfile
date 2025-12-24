# Build Stage
FROM node:20-alpine AS builder

WORKDIR /app

# Install dependencies
COPY package.json package-lock.json ./
RUN npm ci

# Copy source
COPY . .

# Build
RUN npm run build

# Runtime Stage
FROM node:20-alpine

WORKDIR /app

# Copy build artifacts
COPY --from=builder /app/dist ./dist

# Expose port
ENV HOST=0.0.0.0
ENV PORT=4321
EXPOSE 4321

# Start the server
CMD ["node", "./dist/server/entry.mjs"]
