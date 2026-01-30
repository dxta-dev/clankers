#!/usr/bin/env node

import { createPlugin } from '../dist/index.js';

const hookName = process.argv[2];

if (!hookName) {
  console.error('[clankers] No hook name provided');
  process.exit(1);
}

let inputData = '';
process.stdin.setEncoding('utf8');

process.stdin.on('data', (chunk) => {
  inputData += chunk;
});

process.stdin.on('end', async () => {
  try {
    const event = JSON.parse(inputData);

    const plugin = createPlugin();

    if (!plugin) {
      console.error('[clankers] Plugin creation returned null');
      process.exit(0);
    }

    const handler = plugin[hookName];

    if (typeof handler !== 'function') {
      console.error(`[clankers] No handler for hook: ${hookName}`);
      process.exit(0);
    }

    await handler(event);

  } catch (err) {
    console.error(`[clankers] Error in ${hookName}:`, err.message);
  }

  process.exit(0);
});

setTimeout(() => {
  if (!inputData) {
    console.error('[clankers] No input received');
    process.exit(0);
  }
}, 5000);
