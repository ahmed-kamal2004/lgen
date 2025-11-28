// server.js
const grpc = require("@grpc/grpc-js");
const protoLoader = require("@grpc/proto-loader");
const path = require("path");
const fs = require("fs");

// Load proto files
const PROTO_PATH = path.join(__dirname, "protos", "services.proto");
const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
  keepCase: true,
  longs: String,
  enums: String,
  defaults: true,
  oneofs: true,
});

const protoDescriptor = grpc.loadPackageDefinition(packageDefinition);
const chatProto = protoDescriptor.chat;

// In-memory storage for demo purposes
const users = new Map(); // userId -> { username, status }
const chatSessions = new Map(); // sessionId -> Set of call objects
const messageHistory = []; // Store recent messages

// ==============================================
// 1. UNARY RPC - Simple Request/Response
// ==============================================
function sendMessage(call, callback) {
  try {
    const { user_id, username, message, timestamp } = call.request;

    // Validate input
    if (!user_id || !username || !message) {
      return callback({
        code: grpc.status.INVALID_ARGUMENT,
        details: "Missing required fields",
      });
    }

    // Store message
    const messageData = {
      message_id: `msg_${Date.now()}_${Math.random()
        .toString(36)
        .substr(2, 9)}`,
      user_id,
      username,
      message,
      timestamp: timestamp || new Date().toISOString(),
      status: "delivered",
    };

    messageHistory.push(messageData);

    // Keep only last 100 messages
    if (messageHistory.length > 100) {
      messageHistory.shift();
    }

    console.log(`[UNARY] Message received from ${username}: ${message}`);

    // Send success response
    callback(null, {
      message_id: messageData.message_id,
      status: "delivered",
      timestamp: messageData.timestamp,
    });
  } catch (error) {
    console.error("[UNARY ERROR]", error);
    callback({
      code: grpc.status.INTERNAL,
      details: error.message,
    });
  }
}

// ==============================================
// 2. SERVER STREAMING - Push Notifications
// ==============================================
function getNotifications(call) {
    console.log(`[SERVER STREAMING] Client connected, starting flood`);

    let count = 0;

    // Keep sending notifications as fast as possible
    const sendNotification = () => {
        try {
            const notification = {
                notification_id: `notif_${Date.now()}_${count}`,
                type: ["message","system","alert"][Math.floor(Math.random()*3)],
                title: `Notification ${count}`,
                content: `Flood message number ${count}`,
                timestamp: new Date().toISOString(),
                priority: Math.random() > 0.7 ? "high" : "normal"
            };

            call.write(notification);
            count++;

            // Schedule next send immediately (non-blocking)
            setImmediate(sendNotification);

        } catch (err) {
            console.error("[STREAM ERROR]", err);
            call.end();
        }
    };

    // Start sending
    sendNotification();

    // Handle client disconnect
    call.on("cancelled", () => {
        console.log("[SERVER STREAMING] Client disconnected, stopping flood");
    });
}


// ==============================================
// 3. CLIENT STREAMING - File Upload / Batch Processing
// ==============================================
function uploadFile(call, callback) {
  console.log("[CLIENT STREAMING] File upload started");

  let chunks = [];
  let totalBytes = 0;
  let fileMetadata = null;

  call.on("data", (chunk) => {
    if (chunk.chunk_data && chunk.chunk_data.length > 0) {
      chunks.push(chunk.chunk_data);
      totalBytes += chunk.chunk_data.length;
      console.log(
        `[CLIENT STREAMING] Recieved: ${totalBytes} bytes)`
      );
      }
    }
  );

  call.on("end", () => {
    try {
      // Combine all chunks
      const UPLOAD_DIR = "/home/ahmed-kamal/Every/load_generator";
      const completeFile = Buffer.concat(chunks);
      const filePath = path.join(UPLOAD_DIR, `uploaded_${Date.now()}.bin`);
      fs.writeFileSync(filePath, completeFile);

console.log("[CLIENT STREAMING] File saved to:", filePath);

      console.log(
        `[CLIENT STREAMING] Upload complete: ${totalBytes} bytes received`
      );

      callback(null, {
        file_id: `file_${Date.now()}`,
        filename: fileMetadata?.filename || "unknown",
        size: totalBytes,
        status: "success",
        message: "File uploaded successfully",
        upload_time: new Date().toISOString(),
      });
    } catch (error) {
      console.error("[CLIENT STREAMING ERROR]", error);
      callback({
        code: grpc.status.INTERNAL,
        details: error.message,
      });
    }
  });

  call.on("error", (error) => {
    console.error("[CLIENT STREAMING ERROR]", error);
  });
}

// ==============================================
// 4. BIDIRECTIONAL STREAMING - Real-time Chat
// ==============================================
function liveChat(call) {
  let sessionId = null;
  let userId = null;
  let username = null;

  console.log("[BIDIRECTIONAL] New chat connection established");

  call.on("data", (message) => {
    try {
      const {
        session_id,
        user_id,
        username: uname,
        message: msg,
        message_type,
      } = message;

      // Initialize session
      if (!sessionId) {
        sessionId = session_id;
        userId = user_id;
        username = uname;

        if (!chatSessions.has(sessionId)) {
          chatSessions.set(sessionId, new Set());
        }
        chatSessions.get(sessionId).add(call);

        console.log(
          `[BIDIRECTIONAL] User ${username} joined session ${sessionId}`
        );

        // Send welcome message
        call.write({
          message_id: `sys_${Date.now()}`,
          session_id: sessionId,
          user_id: "system",
          username: "System",
          message: `Welcome ${username}! You've joined the chat.`,
          message_type: "system",
          timestamp: new Date().toISOString(),
        });
      }

      if (message_type === "ping") {
        // Respond to ping
        call.write({
          message_id: `pong_${Date.now()}`,
          session_id: sessionId,
          user_id: "system",
          username: "System",
          message: "pong",
          message_type: "pong",
          timestamp: new Date().toISOString(),
        });
        return;
      }

      // Broadcast message to all users in session
      if (msg && chatSessions.has(sessionId)) {
        const broadcastMessage = {
          message_id: `msg_${Date.now()}`,
          session_id: sessionId,
          user_id: userId,
          username: username,
          message: msg,
          message_type: message_type || "text",
          timestamp: new Date().toISOString(),
        };

        console.log(`[BIDIRECTIONAL] Broadcasting from ${username}: ${msg}`);

        // Send to all participants
        chatSessions.get(sessionId).forEach((participantCall) => {
          try {
            participantCall.write(broadcastMessage);
          } catch (error) {
            console.error("[BIDIRECTIONAL] Error broadcasting:", error);
          }
        });
      }
    } catch (error) {
      console.error("[BIDIRECTIONAL ERROR]", error);
    }
  });

  call.on("end", () => {
    if (sessionId && chatSessions.has(sessionId)) {
      chatSessions.get(sessionId).delete(call);
      console.log(`[BIDIRECTIONAL] User ${username} left session ${sessionId}`);

      // Notify others
      chatSessions.get(sessionId).forEach((participantCall) => {
        try {
          participantCall.write({
            message_id: `sys_${Date.now()}`,
            session_id: sessionId,
            user_id: "system",
            username: "System",
            message: `${username} has left the chat`,
            message_type: "system",
            timestamp: new Date().toISOString(),
          });
        } catch (error) {
          console.error("[BIDIRECTIONAL] Error notifying:", error);
        }
      });

      // Clean up empty sessions
      if (chatSessions.get(sessionId).size === 0) {
        chatSessions.delete(sessionId);
      }
    }
    call.end();
  });

  call.on("error", (error) => {
    console.error("[BIDIRECTIONAL ERROR]", error);
  });

  call.on("cancelled", () => {
    console.log("[BIDIRECTIONAL] Client cancelled connection");
  });
}

// ==============================================
// SERVER SETUP
// ==============================================
function main() {
  const server = new grpc.Server({
    "grpc.max_receive_message_length": 10 * 1024 * 1024, // 10MB
    "grpc.max_send_message_length": 10 * 1024 * 1024,
    "grpc.keepalive_time_ms": 30000,
    "grpc.keepalive_timeout_ms": 10000,
    "grpc.http2.max_pings_without_data": 0,
    "grpc.keepalive_permit_without_calls": 1,
  });

  // Register services
  server.addService(chatProto.ChatService.service, {
    SendMessage: sendMessage,
    GetNotifications: getNotifications,
    UploadFile: uploadFile,
    LiveChat: liveChat,
  });

  const port = process.env.PORT || 50051;
  server.bindAsync(
    `0.0.0.0:${port}`,
    grpc.ServerCredentials.createInsecure(),
    (error, port) => {
      if (error) {
        console.error("Failed to start server:", error);
        return;
      }
      console.log(`\n${"=".repeat(60)}`);
      console.log(`🚀 gRPC Server running on port ${port}`);
      console.log(`${"=".repeat(60)}`);
      console.log(`\n📡 Available Services:`);
      console.log(`   1. SendMessage (Unary RPC)`);
      console.log(`   2. GetNotifications (Server Streaming)`);
      console.log(`   3. UploadFile (Client Streaming)`);
      console.log(`   4. LiveChat (Bidirectional Streaming)`);
      console.log(`\n${"=".repeat(60)}\n`);
    }
  );

  // Graceful shutdown
  process.on("SIGINT", () => {
    console.log("\n\nShutting down gRPC server...");
    server.tryShutdown(() => {
      console.log("Server shut down successfully");
      process.exit(0);
    });
  });
}

// Start server
main();
