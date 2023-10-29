use std::{
    collections::HashMap,
    net::TcpListener,
    net::TcpStream,
    sync::{Arc, RwLock},
    io::{Read,Write},
};

const MAX_CLIENTS: usize = 1000;
const MAX_NICK_LEN: usize = 32;

struct Client {
    conn: TcpStream,
    nick: String,
}

impl Clone for Client {
    fn clone(&self) -> Self {
        Self {
            conn: self.conn.try_clone().unwrap(),
            nick: self.nick.clone(),
        }
    }
}

struct ChatState {
    listener: TcpListener,
    clients: Arc<RwLock<HashMap<u16, Client>>>,
    num_clients: usize,
}

impl Clone for ChatState {
    fn clone(&self) -> Self {
        Self {
            listener: self.listener.try_clone().unwrap(),
            clients: self.clients.clone(),
            num_clients: self.num_clients,
        }
    }
}

fn main() {
    let port = 8972;
    let chat_state = ChatState {
        listener: TcpListener::bind(format!("0.0.0.0:{}", port)).unwrap(),
        clients: Arc::new(RwLock::new(HashMap::new())),
        num_clients: 0,
    };

    for stream in chat_state.listener.incoming() {
        let stream = stream.unwrap();
        let port = stream.peer_addr().unwrap().port();
        let client = Client {
            conn: stream,
            nick: format!("user{}", port),
        };

        let mut chat_state = chat_state.clone();
        chat_state.clients.write().unwrap().insert(port, client.clone());
        chat_state.num_clients += 1;


        std::thread::spawn(move || handle_client(client, &mut chat_state));
    }
}

fn handle_client(mut client: Client, chat_state: &mut ChatState) {
    let welcome = "Welcome Simple Chat! Use /nick to change nick name.\n";
    client.conn.write_all(welcome.as_bytes()).unwrap();

    let mut buf = [0; 256];
    loop {
        let n = match client.conn.read(&mut buf) {
            Ok(n) => n,
            Err(_) => {
                println!("{} disconnected", client.nick);
                chat_state.clients.write().unwrap().remove(&client.conn.peer_addr().unwrap().port());
                chat_state.num_clients -= 1;
                return;
            },
        };
            
        let msg = std::str::from_utf8(&buf[..n]).unwrap();
        let msg = msg.trim();

        if msg.starts_with('/') {
            let parts: Vec<_> = msg.splitn(2, ' ').collect();
            match parts[0] {
                "/nick" if parts.len() > 1 => client.nick = parts[1].to_string(),
                _ => {}
            }
            continue;
        }

        println!("{}: {}", client.nick, msg);
        println!("num_clients: {}, {}", chat_state.num_clients, chat_state.clients.read().unwrap().len());

        let mut clients = chat_state.clients.write().unwrap();
        for (_conn, other_client) in clients.iter_mut() {
            if other_client.nick != client.nick {
                other_client
                    .conn
                    .write_all(format!("{}: {}", client.nick, msg).as_bytes())
                    .unwrap();
            }
        }
    }
}
