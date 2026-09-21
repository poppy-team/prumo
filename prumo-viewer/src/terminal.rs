use std::io::{Read, Write};
use std::sync::mpsc::{channel, Receiver, Sender};
use std::sync::{Arc, Mutex};
use portable_pty::{native_pty_system, CommandBuilder, MasterPty, PtySize};
use vt100::Parser;

pub struct LazyTerminal {
    pty_master: Option<Box<dyn MasterPty + Send>>,
    parser: Arc<Mutex<Parser>>,
    writer_tx: Option<Sender<Vec<u8>>>,
    pub is_initialized: bool,
}

impl Default for LazyTerminal {
    fn default() -> Self {
        Self {
            pty_master: None,
            parser: Arc::new(Mutex::new(Parser::new(24, 80, 1000))),
            writer_tx: None,
            is_initialized: false,
        }
    }
}

impl LazyTerminal {
    pub fn new() -> Self {
        Self::default()
    }

    /// Spawns the PTY ONLY on first explicit demand!
    pub fn ensure_spawned(&mut self) -> Result<(), String> {
        if self.is_initialized {
            return Ok(());
        }

        let pty_system = native_pty_system();
        let pair = pty_system
            .openpty(PtySize {
                rows: 24,
                cols: 80,
                pixel_width: 0,
                pixel_height: 0,
            })
            .map_err(|e| format!("openpty error: {}", e))?;

        let shell = std::env::var("SHELL").unwrap_or_else(|_| "/bin/bash".to_string());
        let cmd = CommandBuilder::new(shell);

        let _child = pair
            .slave
            .spawn_command(cmd)
            .map_err(|e| format!("spawn shell error: {}", e))?;

        let mut reader = pair
            .master
            .try_clone_reader()
            .map_err(|e| format!("try_clone_reader error: {}", e))?;

        let mut writer = pair
            .master
            .take_writer()
            .map_err(|e| format!("take_writer error: {}", e))?;

        let parser_clone = Arc::clone(&self.parser);
        std::thread::spawn(move || {
            let mut buf = [0u8; 4096];
            while let Ok(n) = reader.read(&mut buf) {
                if n == 0 {
                    break;
                }
                let mut parser = parser_clone.lock().unwrap();
                parser.process(&buf[..n]);
            }
        });

        let (tx, rx): (Sender<Vec<u8>>, Receiver<Vec<u8>>) = channel();
        std::thread::spawn(move || {
            while let Ok(data) = rx.recv() {
                let _ = writer.write_all(&data);
                let _ = writer.flush();
            }
        });

        self.pty_master = Some(pair.master);
        self.writer_tx = Some(tx);
        self.is_initialized = true;
        Ok(())
    }

    pub fn write_input(&mut self, text: &str) {
        if let Some(tx) = &self.writer_tx {
            let _ = tx.send(text.as_bytes().to_vec());
        }
    }

    pub fn screen_text(&self) -> String {
        let parser = self.parser.lock().unwrap();
        let screen = parser.screen();
        let mut out = String::new();
        for row in 0..screen.size().0 {
            let mut line = String::new();
            for col in 0..screen.size().1 {
                if let Some(cell) = screen.cell(row, col) {
                    line.push(cell.contents().chars().next().unwrap_or(' '));
                }
            }
            out.push_str(line.trim_end());
            out.push('\n');
        }
        out
    }
}
