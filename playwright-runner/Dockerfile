FROM node:23-bookworm-slim AS base

WORKDIR /app

RUN apt-get update && apt-get install -y \
    libnss3 \
    libatk-bridge2.0-0 \
    libx11-xcb1 \
    libxcomposite1 \
    libxdamage1 \
    libxrandr2 \
    libgbm1 \
    libgtk-3-0 \
    libasound2 \
    libxshmfence1 \
    libdrm2 \
    libxfixes3 \
    libxext6 \
    libxrender1 \
    fonts-liberation \
    ca-certificates \
    wget \
    && rm -rf /var/lib/apt/lists/*


FROM base AS build

# Copy workspace root manifests
COPY package.json package-lock.json ./

# Copy workspace packages needed by the runner
COPY packages/synthetics/ ./packages/synthetics/
COPY runner/package.json ./runner/

RUN npm ci --workspace=runner --workspace=@the-grid/synthetics --include-workspace-root

RUN npm run build -w @the-grid/synthetics

RUN npx playwright install chromium


FROM base AS runtime

WORKDIR /app

COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/packages/synthetics/package.json ./packages/synthetics/package.json
COPY --from=build /app/packages/synthetics/dist ./packages/synthetics/dist
COPY --from=build /root/.cache/ms-playwright /root/.cache/ms-playwright
COPY runner/playwright.config.ts .

CMD ["npx", "playwright", "test"]
