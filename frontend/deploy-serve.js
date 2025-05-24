#!/usr/bin/env node

/**
 * Deployment script for serving the React app using 'serve'
 * This can be used for deployment platforms that support Node.js apps
 * or for testing the production build locally.
 */

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

const buildDir = path.join(__dirname, 'build');
const port = process.env.PORT || 3000;

// Check if build directory exists
if (!fs.existsSync(buildDir)) {
  console.error('❌ Build directory not found. Please run "npm run build" first.');
  process.exit(1);
}

console.log(`🚀 Starting server on port ${port}...`);
console.log(`📁 Serving from: ${buildDir}`);

// Start the serve process
const serveProcess = spawn('npx', [
  'serve',
  '-s',        // Single Page Application mode (serves index.html for all non-file requests)
  'build',     // Directory to serve
  '-l',        // Listen on port
  port.toString(),
  '--cors'     // Enable CORS for API requests
], {
  stdio: 'inherit',
  cwd: __dirname
});

// Handle process termination
process.on('SIGINT', () => {
  console.log('\n🛑 Shutting down server...');
  serveProcess.kill('SIGINT');
  process.exit(0);
});

process.on('SIGTERM', () => {
  console.log('\n🛑 Shutting down server...');
  serveProcess.kill('SIGTERM');
  process.exit(0);
});

serveProcess.on('close', (code) => {
  if (code !== 0) {
    console.error(`❌ Server process exited with code ${code}`);
    process.exit(code);
  }
});

serveProcess.on('error', (err) => {
  console.error('❌ Failed to start server:', err);
  process.exit(1);
}); 