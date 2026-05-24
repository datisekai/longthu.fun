# Frontend dev Dockerfile — used by docker-compose for parity with prod.
# Local dev usually runs `pnpm dev` directly on the host (faster HMR);
# this image exists so `docker compose --profile full up` works end-to-end.
FROM node:20-alpine

WORKDIR /app

RUN corepack enable && corepack prepare pnpm@10.9.0 --activate

COPY package.json pnpm-lock.yaml* ./
RUN pnpm install --frozen-lockfile || pnpm install

COPY . .

EXPOSE 5173

CMD ["pnpm", "dev"]
