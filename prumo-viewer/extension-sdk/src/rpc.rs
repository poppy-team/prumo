use serde::{Deserialize, Serialize};
use serde_json::Value;
use std::io::{self, BufRead, Write};

pub const MAX_MESSAGE_BYTES: usize = 1024 * 1024;

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct RpcRequest {
    pub jsonrpc: String,
    pub id: u64,
    pub method: String,
    #[serde(default)]
    pub params: Value,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct RpcResponse {
    pub jsonrpc: String,
    pub id: u64,
    #[serde(default)]
    pub result: Option<Value>,
    #[serde(default)]
    pub error: Option<RpcError>,
}

#[derive(Debug, Clone, Serialize, Deserialize, PartialEq)]
pub struct RpcError {
    pub code: i64,
    pub message: String,
    #[serde(default)]
    pub data: Option<Value>,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum RpcMessage {
    Request,
    Response,
}

pub trait StdioExtension {
    fn handle(&mut self, method: &str, params: Value) -> Result<Value, RpcError>;
}

pub fn write_message<W: Write>(writer: &mut W, value: &impl Serialize) -> Result<(), String> {
    let mut bytes = serde_json::to_vec(value).map_err(|error| error.to_string())?;
    if bytes.len() > MAX_MESSAGE_BYTES {
        return Err("RPC message exceeds maximum size".to_string());
    }
    bytes.push(b'\n');
    writer.write_all(&bytes).map_err(|error| error.to_string())
}

pub fn read_message<R: BufRead>(reader: &mut R) -> Result<Option<Value>, String> {
    let mut bytes = Vec::new();
    loop {
        let available = reader.fill_buf().map_err(|error| error.to_string())?;
        if available.is_empty() {
            break;
        }
        let newline = available.iter().position(|byte| *byte == b'\n');
        let take = newline.unwrap_or(available.len());
        if bytes.len().saturating_add(take) > MAX_MESSAGE_BYTES {
            return Err("RPC message exceeds maximum size".to_string());
        }
        bytes.extend_from_slice(&available[..take]);
        reader.consume(take + usize::from(newline.is_some()));
        if newline.is_some() {
            break;
        }
    }
    if bytes.is_empty() {
        return Ok(None);
    }
    serde_json::from_slice(&bytes)
        .map(Some)
        .map_err(|error| error.to_string())
}

pub fn serve_stdio(mut extension: impl StdioExtension) -> Result<(), String> {
    let stdin = io::stdin();
    let mut input = stdin.lock();
    let stdout = io::stdout();
    let mut output = stdout.lock();
    while let Some(message) = read_message(&mut input)? {
        let value = message;
        let request = match serde_json::from_value::<RpcRequest>(value.clone()) {
            Ok(request) => request,
            Err(response_error) => match serde_json::from_value::<RpcResponse>(value) {
                Ok(_) => continue,
                Err(_) => {
                    return Err(format!("invalid RPC message: {response_error}"));
                }
            },
        };
        let result = extension.handle(&request.method, request.params);
        let response = match result {
            Ok(result) => RpcResponse {
                jsonrpc: "2.0".to_string(),
                id: request.id,
                result: Some(result),
                error: None,
            },
            Err(error) => RpcResponse {
                jsonrpc: "2.0".to_string(),
                id: request.id,
                result: None,
                error: Some(error),
            },
        };
        write_message(&mut output, &response)?;
    }
    Ok(())
}

impl From<String> for RpcError {
    fn from(message: String) -> Self {
        Self {
            code: -32000,
            message,
            data: None,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Cursor;

    #[test]
    fn reads_bounded_json_lines() {
        let mut input = Cursor::new(b"{\"id\":1}\n{\"id\":2}\n".as_slice());
        assert_eq!(
            read_message(&mut input).unwrap(),
            Some(serde_json::json!({"id": 1}))
        );
        assert_eq!(
            read_message(&mut input).unwrap(),
            Some(serde_json::json!({"id": 2}))
        );
        assert_eq!(read_message(&mut input).unwrap(), None);
    }

    #[test]
    fn rejects_oversized_messages() {
        let payload = vec![b'x'; MAX_MESSAGE_BYTES + 1];
        let mut input = Cursor::new(payload);
        assert!(read_message(&mut input).is_err());
    }
}
