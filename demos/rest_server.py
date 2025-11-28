# rest_server_corrected.py
from fastapi import FastAPI, WebSocket, WebSocketDisconnect, Request, HTTPException, status, Query
from fastapi.responses import StreamingResponse
from pydantic import BaseModel, Field, validator
from typing import List, Optional, Set
import uuid
import json
import asyncio
import os
from datetime import datetime, timezone
import base64

DATA_FILE = "data.json"
app = FastAPI(title="REST Chat Server Imitating gRPC Services")

# # RESET STORE ON STARTUP (used it for testing)
# with open(DATA_FILE, "w") as f:
#     json.dump({"messages": [], "notifications": []}, f, indent=2)


# ======================================================
# Helpers
# ======================================================
def ensure_store():
    if not os.path.exists(DATA_FILE):
        with open(DATA_FILE, "w") as f:
            json.dump({"messages": [], "notifications": []}, f, indent=2)


def load_store():
    ensure_store()
    with open(DATA_FILE, "r") as f:
        return json.load(f)


def save_store(data):
    with open(DATA_FILE, "w") as f:
        json.dump(data, f, indent=2)


def utc_now_iso():
    return datetime.now(timezone.utc).isoformat()


# ======================================================
# MODELS (MATCH EXACT PROTO STRUCTURE)
# ======================================================

# ----- SendMessage (Unary) -----
class MessageRequest(BaseModel):
    user_id: str
    username: str
    message: str
    timestamp: str


class MessageResponse(BaseModel):
    message_id: str
    status: str
    timestamp: str


# ----- Notification Streaming -----
class NotificationRequest(BaseModel):
    user_id: str
    notification_types: Optional[List[str]] = []


class Notification(BaseModel):
    notification_id: str
    user_id: str
    type: str  # message, system, alert
    title: str
    content: str
    timestamp: str
    priority: str  # low, normal, high


# ----- ChatMessage (Bidirectional) -----
class ChatMessage(BaseModel):
    message_id: Optional[str] = None
    session_id: str
    user_id: str
    username: str
    message: str
    message_type: str  # text, image, file, system, ping, pong
    timestamp: str
    attachments: Optional[List[str]] = []

    @validator("message_type")
    def validate_message_type(cls, v):
        allowed = {"text", "image", "file", "system", "ping", "pong"}
        if v not in allowed:
            raise ValueError(f"message_type must be one of {allowed}")
        return v

    @validator("attachments", pre=True, always=True)
    def ensure_attachments(cls, v):
        return v or []


# ----- UploadFile (Client streaming imitation) -----
class FileMetadata(BaseModel):
    filename: str
    file_type: str
    file_size: int
    user_id: str


class FileChunk(BaseModel):
    metadata: Optional[FileMetadata] = None
    chunk_data: str  # base64 string expected
    chunk_number: int
    is_last_chunk: bool

    @validator("chunk_number")
    def chunk_number_positive(cls, v):
        if v < 1:
            raise ValueError("chunk_number must be >= 1")
        return v


class FileUploadResponse(BaseModel):
    file_id: str
    filename: str
    size: int
    status: str
    message: str
    upload_time: str


# ======================================================
# Internal: Store ChatMessage + Notification
# ======================================================
def store_message(msg: ChatMessage):
    data = load_store()

    msg_id = msg.message_id or str(uuid.uuid4())
    saved_msg = {
        "message_id": msg_id,
        "session_id": msg.session_id,
        "user_id": msg.user_id,
        "username": msg.username,
        "message": msg.message,
        "message_type": msg.message_type,
        "attachments": msg.attachments or [],
        "timestamp": msg.timestamp,
    }

    data["messages"].append(saved_msg)

    notif = {
        "notification_id": str(uuid.uuid4()),
        "user_id": msg.user_id,
        "type": "message",
        "title": f"New message from {msg.username}",
        "content": msg.message[:200],
        "timestamp": msg.timestamp,
        "priority": "normal",
    }

    data["notifications"].append(notif)
    save_store(data)

    return saved_msg


# ======================================================
# 1. UNARY RPC → /SendMessage
# ======================================================
@app.post("/SendMessage", response_model=MessageResponse, status_code=status.HTTP_200_OK)
async def send_message(req: MessageRequest):
    """
    Unary RPC equivalent. The proto MessageRequest doesn't include message_type,
    so we persist a ChatMessage with message_type="text".
    """
    # validate timestamp format? Assume caller provides proper ISO timestamp
    chat_msg = ChatMessage(
        message_id=str(uuid.uuid4()),
        session_id="default",
        user_id=req.user_id,
        username=req.username,
        message=req.message,
        message_type="text",
        timestamp=req.timestamp,
        attachments=[]
    )

    saved = store_message(chat_msg)

    return MessageResponse(
        message_id=saved["message_id"],
        status="OK",
        timestamp=saved["timestamp"]
    )


# ======================================================
# 2. SERVER STREAMING → /GetNotifications (SSE) with filtering
# ======================================================
@app.get("/GetNotifications")
async def get_notifications(request: Request, user_id: str, notification_types: Optional[str] = Query(None, description="Comma-separated list of notification types to filter by (e.g. 'message,alert')")):
    """
    SSE endpoint.
    Query params:
      - user_id
      - notification_types (optional, comma separated)
    """
    # Parse notification_types into a set for fast lookup (if provided)
    types_set: Optional[Set[str]] = None
    if notification_types:
        types_set = {t.strip() for t in notification_types.split(",") if t.strip()}

    return StreamingResponse(
        notifications_stream(request, user_id, types_set),
        media_type="text/event-stream"
    )


async def notifications_stream(request: Request, user_id: str, types_set: Optional[Set[str]] = None):
    """
    Polls the store and yields new notifications for the user
    optionally filtered by types_set.
    """

    while True:
        if await request.is_disconnected():
            break

        store = load_store()

        for n in store["notifications"]:

            payload = (
                "event: notification\n"
                f"data: {json.dumps(n)}\n\n"
            )
            
            yield payload.encode()
            await asyncio.sleep(0)


# ======================================================
# 3. CLIENT STREAMING → /UploadFile (REST imitation)
# ======================================================
# @app.post("/UploadFile", response_model=FileUploadResponse)
# async def upload_file(chunks: List[FileChunk]):
#     """
#     Accepts a list (batch) of FileChunk objects which mimic a client-streamed upload.
#     Requirements & validation:
#       - The first chunk must include metadata (FileMetadata).
#       - chunk_data must be base64-encoded.
#       - chunk_number must start at 1 and increment by 1 for each chunk (strict order).
#       - is_last_chunk must be True on the final chunk.
#       - The total bytes received must match metadata.file_size (if provided).
#     """
#     if not chunks:
#         raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="No chunks received")

#     first = chunks[0]

#     if first.metadata is None:
#         raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="Missing metadata in the first chunk")

#     # Validate metadata presence only in first chunk
#     for i, ch in enumerate(chunks):
#         if i != 0 and ch.metadata is not None:
#             raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="Metadata must only be provided in the first chunk")

#     # Validate chunk numbers are consecutive starting from 1
#     expected = 1
#     for ch in chunks:
#         if ch.chunk_number != expected:
#             raise HTTPException(
#                 status_code=status.HTTP_400_BAD_REQUEST,
#                 detail=f"Invalid chunk sequence: expected chunk_number {expected} but got {ch.chunk_number}"
#             )
#         expected += 1

#     # The last chunk must have is_last_chunk=True
#     if not chunks[-1].is_last_chunk:
#         raise HTTPException(status_code=status.HTTP_400_BAD_REQUEST, detail="Final chunk must have is_last_chunk=True")

#     total_bytes = 0
#     all_data = bytearray()

#     for ch in chunks:
#         try:
#             data_bytes = base64.b64decode(ch.chunk_data, validate=True)
#         except Exception as e:
#             raise HTTPException(status_code=400, detail=f"chunk_data must be valid base64: {e}")

#         all_data.extend(data_bytes)
#         total_bytes += len(data_bytes)

#     # Validate declared file_size BEFORE writing to disk
#     declared_size = first.metadata.file_size
#     if declared_size != total_bytes:
#         raise HTTPException(
#             status_code=400,
#             detail=f"Declared file_size {declared_size} does not match received size {total_bytes}"
#         )

#     # --- Only write to disk AFTER every validation passed ---
#     file_id = str(uuid.uuid4())
#     safe_filename = first.metadata.filename.replace("/", "_").replace("..", "_")
#     file_path = f"uploaded_{file_id}_{safe_filename}"

#     try:
#         with open(file_path, "wb") as f:
#             f.write(all_data)
#     except Exception as e:
#         raise HTTPException(status_code=500, detail=f"Failed to write file: {e}")

#     return FileUploadResponse(
#         file_id=file_id,
#         filename=first.metadata.filename,
#         size=total_bytes,
#         status="OK",
#         message="File uploaded successfully",
#         upload_time=datetime.utcnow().isoformat()
#     )


UPLOAD_DIR = "/home/ahmed-kamal/Every/load_generator"

@app.post("/upload")
async def upload_file(request: Request):
    """
    Handles chunked transfer encoding uploads.
    Reads the request body as a stream.
    """
    file_id = str(uuid.uuid4())
    file_path = UPLOAD_DIR + "/" + f"uploaded_{file_id}.bin"

    try:
        with open(file_path, "wb") as f:
            async for chunk in request.stream():
                print(f"Received chunk of {len(chunk)} bytes")
                f.write(chunk)
    except Exception as e:
        print("error", e)
        raise HTTPException(status_code=500, detail=f"Failed to write file: {e}")

    return {"file_id": file_id, "status": "ok"}


# ======================================================
# 4. BIDIRECTIONAL STREAM → /LiveChat WebSocket
# ======================================================
class WSManager:
    def __init__(self):
        self.active: List[WebSocket] = []

    async def connect(self, ws: WebSocket):
        await ws.accept()
        self.active.append(ws)

    def remove(self, ws: WebSocket):
        if ws in self.active:
            self.active.remove(ws)

    async def broadcast(self, message: str):
        """
        Broadcast a JSON string to all active websockets.
        If a socket fails, remove it.
        """
        for ws in list(self.active):
            try:
                await ws.send_text(message)
            except Exception:
                self.remove(ws)


ws_mgr = WSManager()


@app.websocket("/LiveChat")
async def websocket_chat(ws: WebSocket):
    """
    WebSocket endpoint that accepts and returns ChatMessage JSON objects
    matching the proto ChatMessage structure.
    """
    await ws_mgr.connect(ws)
    try:
        while True:
            raw = await ws.receive_text()
            try:
                data = json.loads(raw)
            except json.JSONDecodeError:
                # send an error back to the client (protocol-level)
                await ws.send_text(json.dumps({"error": "invalid json"}))
                continue

            # Validate incoming payload as ChatMessage (will raise if invalid)
            try:
                # If message_type not present, default to 'text' (but validator enforces allowed values)
                if "message_type" not in data:
                    data["message_type"] = "text"

                # If timestamp missing, add one
                if "timestamp" not in data or not data["timestamp"]:
                    data["timestamp"] = utc_now_iso()

                msg = ChatMessage(**data)
            except Exception as e:
                await ws.send_text(json.dumps({"error": f"invalid ChatMessage: {str(e)}"}))
                continue

            # Ensure message_id exists
            if not msg.message_id:
                msg.message_id = str(uuid.uuid4())

            # Persist the message (store_message expects ChatMessage object)
            saved = store_message(msg)

            # Broadcast the full proto-style ChatMessage to all clients
            broadcast_msg = {
                "message_id": msg.message_id,
                "session_id": msg.session_id,
                "user_id": msg.user_id,
                "username": msg.username,
                "message": msg.message,
                "message_type": msg.message_type,
                "timestamp": msg.timestamp,
                "attachments": msg.attachments or []
            }

            await ws_mgr.broadcast(json.dumps(broadcast_msg))

    except WebSocketDisconnect:
        ws_mgr.remove(ws)
    except Exception:
        # ensure we remove socket on unexpected errors
        ws_mgr.remove(ws)


# # ======================================================
# # Debug Endpoints (not in proto, for testing only)
# # ======================================================
# @app.get("/_admin/messages")
# async def admin_messages():
#     return load_store()["messages"]


# @app.get("/_admin/notifications")
# async def admin_notifications():
#     return load_store()["notifications"]
# ======================================================