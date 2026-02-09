#!/bin/bash
set -e

# ─── SWARM C2 — GitHub + Railway Deploy Script ───────────────────────────
# Run this from the swarm-c2 directory on your local machine
# Usage: chmod +x deploy.sh && ./deploy.sh

RED='\033[0;31m'
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${CYAN}"
echo "╔═══════════════════════════════════════════╗"
echo "║        SWARM C2 — DEPLOY SCRIPT           ║"
echo "╚═══════════════════════════════════════════╝"
echo -e "${NC}"

# ─── Pre-flight checks ───────────────────────────────────────────────────
echo -e "${YELLOW}[1/7] Pre-flight checks...${NC}"

command -v git >/dev/null 2>&1 || { echo -e "${RED}✗ git not found. Install git first.${NC}"; exit 1; }
command -v go >/dev/null 2>&1 || { echo -e "${RED}✗ go not found. Install Go 1.21+ first.${NC}"; exit 1; }
command -v node >/dev/null 2>&1 || { echo -e "${RED}✗ node not found. Install Node.js 18+ first.${NC}"; exit 1; }

echo -e "${GREEN}  ✓ git $(git --version | cut -d' ' -f3)${NC}"
echo -e "${GREEN}  ✓ go $(go version | cut -d' ' -f3)${NC}"
echo -e "${GREEN}  ✓ node $(node --version)${NC}"

# ─── Check .env exists ───────────────────────────────────────────────────
if [ ! -f .env ]; then
  echo -e "${YELLOW}  ⚠ No .env file found. Copy .env.example and add your keys:${NC}"
  echo "    cp .env.example .env"
  echo ""
fi

# ─── Generate go.sum ─────────────────────────────────────────────────────
echo -e "${YELLOW}[2/7] Generating Go dependencies...${NC}"
cd backend
go mod tidy
echo -e "${GREEN}  ✓ go.sum generated${NC}"
cd ..

# ─── Install frontend deps ──────────────────────────────────────────────
echo -e "${YELLOW}[3/7] Installing frontend dependencies...${NC}"
cd frontend
npm install --silent
echo -e "${GREEN}  ✓ node_modules ready${NC}"

# ─── Build frontend ─────────────────────────────────────────────────────
echo -e "${YELLOW}[4/7] Building frontend...${NC}"
npm run build --silent
echo -e "${GREEN}  ✓ Frontend built → dist/${NC}"
cd ..

# ─── Test Go build ───────────────────────────────────────────────────────
echo -e "${YELLOW}[5/7] Testing Go build...${NC}"
cd backend
mkdir -p static
cp -r ../frontend/dist/* static/
go build -o swarm-c2
echo -e "${GREEN}  ✓ Backend compiles successfully${NC}"
rm -f swarm-c2  # clean up test binary
rm -rf static   # Railway will do this during deploy
cd ..

# ─── Initialize Git ─────────────────────────────────────────────────────
echo -e "${YELLOW}[6/7] Setting up Git repository...${NC}"

if [ ! -d .git ]; then
  git init
  echo -e "${GREEN}  ✓ Git repo initialized${NC}"
else
  echo -e "${GREEN}  ✓ Git repo already exists${NC}"
fi

# Stage all files
git add .
git status --short

echo ""
echo -e "${CYAN}─── Ready to commit and push ────────────────────────────${NC}"
echo ""

# ─── Prompt for GitHub remote ────────────────────────────────────────────
echo -e "${YELLOW}[7/7] GitHub setup${NC}"
echo ""

REMOTE_EXISTS=$(git remote get-url origin 2>/dev/null || echo "")

if [ -z "$REMOTE_EXISTS" ]; then
  echo -e "  Create a new repository on GitHub:"
  echo -e "  ${CYAN}https://github.com/new${NC}"
  echo -e "  Name: ${GREEN}swarm-c2${NC}"
  echo -e "  Visibility: Private (recommended — contains API key patterns)"
  echo ""
  read -p "  Enter your GitHub repo URL (e.g. https://github.com/user/swarm-c2.git): " REPO_URL

  if [ -n "$REPO_URL" ]; then
    git remote add origin "$REPO_URL"
    echo -e "${GREEN}  ✓ Remote added: $REPO_URL${NC}"
  else
    echo -e "${YELLOW}  ⚠ Skipping remote. Add manually later:${NC}"
    echo "    git remote add origin https://github.com/YOUR_USER/swarm-c2.git"
  fi
else
  echo -e "${GREEN}  ✓ Remote already set: $REMOTE_EXISTS${NC}"
fi

echo ""
echo -e "${CYAN}─── Run these commands to finish: ───────────────────────${NC}"
echo ""
echo -e "  ${GREEN}git commit -m \"SWARM C2: aircraft tracking platform\"${NC}"
echo -e "  ${GREEN}git branch -M main${NC}"
echo -e "  ${GREEN}git push -u origin main${NC}"
echo ""
echo -e "${CYAN}─── Then deploy on Railway: ─────────────────────────────${NC}"
echo ""
echo "  1. Go to https://railway.app → New Project → Deploy from GitHub"
echo "  2. Select your swarm-c2 repo"
echo "  3. Add these variables in the Variables tab:"
echo -e "     ${GREEN}OPENAI_API_KEY${NC}=your-key"
echo -e "     ${GREEN}VITE_MAPTILER_KEY${NC}=your-key"
echo -e "     ${GREEN}OPENSKY_CLIENT_ID${NC}=your-id"
echo -e "     ${GREEN}OPENSKY_CLIENT_SECRET${NC}=your-secret"
echo -e "     ${GREEN}PORT${NC}=8080"
echo ""
echo "  4. Railway auto-detects railway.json and deploys"
echo ""
echo -e "${GREEN}╔═══════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║          DEPLOY PREP COMPLETE ✓           ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════╝${NC}"
