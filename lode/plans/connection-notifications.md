# Plugin Connection Notifications Plan

## Summary
Add user-visible notifications when plugins connect to (or fail to connect to) the clankers daemon on startup.

## Current State

Both plugins currently check daemon health on startup:

### OpenCode Plugin
- **Location**: `apps/opencode-plugin/src/index.ts` lines 590-613
- **Pattern**: Synchronous `await rpc.health()` at plugin init
- **Current behavior**: Shows toast warning when daemon not running
- **Missing**: Success notification when connected

### Claude Code Plugin  
- **Location**: `apps/claude-code-plugin/src/index.ts` lines 291-323
- **Pattern**: Async promise with `waitForConnection()` helper
- **Current behavior**: Logs only (no user-visible notification)
- **Missing**: Both success and failure user notifications

## Goals

1. **Notify on successful connection** - Both plugins should tell users they're connected to clankers
2. **Notify on connection failure** - Both plugins should tell users the daemon isn't running and how to start it
3. **Respect platform capabilities** - Use appropriate notification mechanisms for each platform

## Implementation Approach

### OpenCode Plugin

OpenCode has `client.tui.showToast()` for notifications.

#### Success Notification
```ts
export const ClankersPlugin: Plugin = async ({ client }) => {
	const rpc = createRpcClient({
		clientName: "opencode-plugin",
		clientVersion: "0.1.0",
	});

	let connected = false;
	let connectionError: string | undefined;
	
	try {
		const health = await rpc.health();
		if (health.ok) {
			connected = true;
			logger.info(`Connected to clankers v${health.version}`);
			// NEW: Show success toast
			void client.tui.showToast({
				body: {
					message: `Clankers connected (v${health.version})`,
					variant: "success",
				},
			});
		}
	} catch (error) {
		connectionError = error instanceof Error ? error.message : String(error);
		logger.warn("Clankers daemon not running; events will be skipped", {
			error: connectionError,
		});
		void client.tui.showToast({
			body: {
				message: "Clankers daemon not running. Start it to enable sync.",
				variant: "warning",
			},
		});
	}

	return { /* ... */ };
};
```

**Message format**:
- Success: `"Clankers connected (v1.2.3)"` - variant: `success`
- Failure: `"Clankers daemon not running. Start it to enable sync."` - variant: `warning`

### Claude Code Plugin

Claude Code has no toast API. Use `hookSpecificOutput` context messages via stdout.

#### Success Notification in SessionStart
```ts
export function createPlugin(): ClaudeCodeHooks | null {
	const rpc = createRpcClient({
		clientName: "claude-code-plugin",
		clientVersion: "0.1.0",
	});

	let connectionState: boolean | null = null;
	let connectionPromise: Promise<boolean> | null = null;
	let connectionNotified = false; // NEW: Track if we already notified

	connectionPromise = rpc
		.health()
		.then((health) => {
			connectionState = health.ok;
			if (connectionState) {
				logger.info(`Connected to clankers v${health.version}`);
			}
			return connectionState;
		})
		.catch(() => {
			logger.warn("Daemon not running; events will be skipped");
			connectionState = false;
			return false;
		});

	async function waitForConnection(): Promise<boolean> {
		if (connectionState !== null) return connectionState;
		if (connectionPromise) return connectionPromise;
		return false;
	}

	return {
		SessionStart: async (event: SessionStartEvent) => {
			const connected = await waitForConnection();
			
			// NEW: Notify once per session start
			if (!connectionNotified) {
				connectionNotified = true;
				if (connected) {
					// Success - add to context
					console.log(JSON.stringify({
						hookSpecificOutput: {
							hookEventName: "SessionStart",
							additionalContext: "✅ Clankers connected - sessions will be synced"
						}
					}));
				} else {
					// Failure - add warning to context  
					console.log(JSON.stringify({
						hookSpecificOutput: {
							hookEventName: "SessionStart",
							additionalContext: "⚠️ Clankers daemon not running. Run 'clankers daemon' to enable session sync."
						}
					}));
				}
			}
			
			if (!connected) return;
			// ... rest of handler
		},
		// ... other hooks
	};
}
```

**Message format**:
- Success: `"✅ Clankers connected - sessions will be synced"`
- Failure: `"⚠️ Clankers daemon not running. Run 'clankers daemon' to enable session sync."`

## Key Design Decisions

### 1. Notify Once Per Session
Claude Code hooks run in separate processes, but `SessionStart` only fires once per session. Using `connectionNotified` flag prevents duplicate notifications if the user manually triggers hooks.

### 2. Use SessionStart Hook
`SessionStart` is the first hook fired and represents the beginning of user interaction. This is the natural place to show connection status since:
- User is starting a new conversation
- Context output appears at the beginning
- Failed daemon = no session sync for entire conversation

### 3. Different Message Styles
- **OpenCode**: Concise toast (limited space), version number for debugging
- **Claude Code**: Full sentence with emoji, clear action instruction

### 4. Variant/Severity Levels
- **Success**: Green/success state - daemon connected, all features available
- **Warning**: Yellow/warning state - plugin works but no persistence

## Testing Strategy

### OpenCode
1. Start daemon: verify success toast appears
2. Stop daemon: verify warning toast appears
3. Check toast message content and variant

### Claude Code  
1. Start daemon: verify context output in conversation start
2. Stop daemon: verify warning context with instruction
3. Verify output format matches `hookSpecificOutput` schema

## Files to Modify

| File | Lines | Change |
|------|-------|--------|
| `apps/opencode-plugin/src/index.ts` | 590-613 | Add success toast, capture version |
| `apps/claude-code-plugin/src/index.ts` | 291-330 | Add connectionNotified flag, context output in SessionStart |

## Future Considerations

1. **Retry logic**: Could attempt reconnection in later hooks if initial connection fails
2. **Background sync**: Queue events while disconnected, sync when daemon starts
3. **Configuration**: Allow users to disable connection notifications via settings
4. **Reconnection detection**: Periodically re-check if daemon starts after initial failure

## References

- [OpenCode Plugin](opencode/plugins.md)
- [Claude Code Plugin System](claude/plugin-system.md)
- [RPC Client](../packages/core/src/rpc-client.ts) - `health()` method
