FROM node:22-alpine

WORKDIR /app

COPY package.json package-lock.json tsconfig.base.json ./
COPY packages/contracts ./packages/contracts
COPY exapps/control-plane ./exapps/control-plane

RUN npm install
RUN npm run build

WORKDIR /app/exapps/control-plane

CMD ["npm", "run", "start"]

