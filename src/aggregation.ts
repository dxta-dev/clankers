const syncedMessages = new Set<string>();
const messagePartsText = new Map<string, string[]>();
type MessageInfo = {
  modelID?: string;
  tokens?: { input?: number; output?: number };
  time?: { created?: number; completed?: number };
};

const messageMetadata = new Map<
  string,
  { role: string; sessionId: string; info: Partial<MessageInfo> }
>();
const syncTimeouts = new Map<string, ReturnType<typeof setTimeout>>();
const DEBOUNCE_MS = 800;

export function inferRole(textContent: string): "user" | "assistant" {
	const assistantPatterns = [
		/^(I'll|Let me|Here's|I can|I've|I'm going to|I will|Sure|Certainly|Of course)/i,
		/```[\s\S]+```/,
		/^(Yes|No),?\s+(I|you|we|this|that)/i,
		/\*\*[^*]+\*\*/,
		/^\d+\.\s+\*\*/,
	];
	const userPatterns = [
		/\?$/,
		/^(create|fix|add|update|show|make|build|implement|write|delete|remove|change|modify|help|can you|please|I want|I need)/i,
		/^@/,
	];
	for (const pattern of assistantPatterns) {
		if (pattern.test(textContent)) return "assistant";
	}
	for (const pattern of userPatterns) {
		if (pattern.test(textContent)) return "user";
	}
	return textContent.length > 500 ? "assistant" : "user";
}

export function stageMessageMetadata(info: {
  id?: string;
  sessionID?: string;
  role?: string;
  modelID?: string;
  tokens?: { input?: number; output?: number };
  time?: { created?: number; completed?: number };
}): void {
	if (!info?.id || !info?.sessionID) return;
	messageMetadata.set(info.id, {
		role: info.role || "unknown",
		sessionId: info.sessionID,
		info,
	});
}

export function stageMessagePart(part: {
	type?: string;
	messageID?: string;
	sessionID?: string;
	text?: string;
}): void {
	if (part?.type !== "text" || !part?.messageID || !part?.sessionID) return;
	const messageId = part.messageID;
	const text = part.text || "";
	messagePartsText.set(messageId, [text]);
	if (!messageMetadata.has(messageId)) {
		messageMetadata.set(messageId, {
			role: "unknown",
			sessionId: part.sessionID,
			info: {},
		});
	}
}

export function scheduleMessageFinalize(
  messageId: string,
  onReady: (payload: {
    messageId: string;
    sessionId: string;
    role: string;
    textContent: string;
    info: Partial<MessageInfo>;
  }) => void,
): void {
	const existing = syncTimeouts.get(messageId);
	if (existing) clearTimeout(existing);
	const timeout = setTimeout(() => {
		syncTimeouts.delete(messageId);
		finalizeMessage(messageId, onReady);
	}, DEBOUNCE_MS);
	syncTimeouts.set(messageId, timeout);
}

function finalizeMessage(
  messageId: string,
  onReady: (payload: {
    messageId: string;
    sessionId: string;
    role: string;
    textContent: string;
    info: Partial<MessageInfo>;
  }) => void,
): void {
	if (syncedMessages.has(messageId)) return;
	const metadata = messageMetadata.get(messageId);
	const textParts = messagePartsText.get(messageId);
	if (!metadata || !textParts || textParts.length === 0) return;

	const textContent = textParts.join("");
	if (!textContent.trim()) return;

	syncedMessages.add(messageId);
	onReady({
		messageId,
		sessionId: metadata.sessionId,
		role: metadata.role,
		textContent,
		info: metadata.info,
	});

	messagePartsText.delete(messageId);
	messageMetadata.delete(messageId);
}
