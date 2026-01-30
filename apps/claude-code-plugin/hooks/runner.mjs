#!/usr/bin/env node
// Shell hook runner - bridges Claude Code shell hooks to our TypeScript plugin

import { createPlugin } from '../dist/index.js';

const hookName = process.argv[2];

if (!hookName) {
  console.error('[clankers] No hook name provided');
  process.exit(1);
}

// Read event data from stdin
let inputData = '';
process.stdin.setEncoding('utf8');

process.stdin.on('data', (chunk) => {
  inputData += chunk;
});

process.stdin.on('end', async () => {
  try {
    const event = JSON.parse(inputData);

    // Create plugin instance
    const plugin = createPlugin();

    if (!plugin) {
      console.error('[clankers] Plugin creation returned null');
      process.exit(0);
    }

    // Get the handler for this hook
    const handler = plugin[hookName];

    if (typeof handler !== 'function') {
      console.error(`[clankers] No handler for hook: ${hookName}`);
      process.exit(0);
    }

    // Call the handler
    await handler(event);

  } catch (err) {
    console.error(`[clankers] Error in ${hookName}:`, err.message);
  }

  process.exit(0);
});

// Handle no stdin (shouldn't happen, but be safe)
setTimeout(() => {
  if (!inputData) {
    console.error('[clankers] No input received');
    process.exit(0);
  }
}, 5000);
