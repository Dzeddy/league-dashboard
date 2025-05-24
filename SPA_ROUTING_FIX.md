# SPA Routing Fix: "Uncaught SyntaxError: Unexpected token '<'"

## Quick Fix Summary

The "Uncaught SyntaxError: Unexpected token '<'" error has been resolved by implementing environment-specific asset path configurations.

## What Was Fixed

1. **Root Cause**: The `homepage` field in `package.json` was set to the GitHub Pages URL, causing incorrect asset paths for local development
2. **Impact**: When accessing `http://localhost:3000/league_dashboard`, the browser tried to load JS/CSS from wrong paths, receiving HTML instead of JavaScript files
3. **Solution**: Added scripts to dynamically set the `homepage` field based on deployment target

## Quick Commands

### For Local Development
```bash
cd frontend
npm run build-and-serve
```
This sets `homepage: "."` (relative paths) and serves locally.

### For GitHub Pages Deployment
```bash
cd frontend
npm run deploy
```
This sets `homepage: "https://dzeddy.github.io/league-dashboard"` and deploys.

### Check Current Configuration
```bash
npm pkg get homepage
```

## What Changed

1. **package.json**: Added new scripts for environment-specific builds
2. **DEPLOYMENT.md**: Updated with comprehensive troubleshooting guide
3. **Asset Paths**: Now correctly resolve for both local and production environments

## Verification

The fix allows you to:
- Access `http://localhost:3000/league_dashboard` without errors
- Deploy to GitHub Pages without asset loading issues
- Switch between development and production configurations easily

For detailed information, see `frontend/DEPLOYMENT.md`. 