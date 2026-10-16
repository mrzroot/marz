FROM node:22-alpine AS build
WORKDIR /app
COPY . .
RUN npm ci && npm run build

FROM python:3.12-slim
COPY --from=build /app/dist /srv/static
COPY server /srv
CMD ["python", "-m", "uvicorn", "main:app"]
