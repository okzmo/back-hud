REMOVE TABLE IF EXISTS users;
REMOVE TABLE IF EXISTS sessions;
REMOVE TABLE IF EXISTS friends;
REMOVE TABLE IF EXISTS servers;
REMOVE TABLE IF EXISTS channels;
REMOVE TABLE IF EXISTS messages;
REMOVE TABLE IF EXISTS notifications;
REMOVE TABLE IF EXISTS subscribed;
REMOVE TABLE IF EXISTS member;

-- users
DEFINE TABLE users SCHEMAFULL;

DEFINE FIELD email ON TABLE users TYPE string ASSERT string::is::email($value);
DEFINE FIELD password ON TABLE users TYPE string;
DEFINE FIELD username ON TABLE users TYPE string;
DEFINE FIELD display_name ON TABLE users TYPE string;
DEFINE FIELD avatar ON TABLE users TYPE string;
DEFINE FIELD banner ON TABLE users TYPE string;
DEFINE FIELD created_at ON TABLE users TYPE datetime DEFAULT time::now();
DEFINE FIELD status ON TABLE users TYPE string;
DEFINE FIELD about_me ON TABLE users TYPE string;
DEFINE INDEX idx_email ON TABLE users COLUMNS email UNIQUE;
DEFINE INDEX idx_username ON TABLE users COLUMNS username UNIQUE;

-- sessions
DEFINE TABLE sessions SCHEMAFULL;

DEFINE FIELD user_id ON TABLE sessions TYPE record<users>;
DEFINE FIELD created_at ON TABLE sessions TYPE datetime DEFAULT time::now();
DEFINE FIELD expires_at ON TABLE sessions TYPE datetime DEFAULT time::now() + 30d;
DEFINE FIELD ip_address ON TABLE sessions TYPE string;
DEFINE FIELD user_agent ON TABLE sessions TYPE string;

-- friends relation
DEFINE TABLE friends TYPE RELATION FROM users TO users;
DEFINE FIELD accepted ON TABLE friends TYPE bool DEFAULT false;
DEFINE INDEX unique_relationships
        ON TABLE friends
        COLUMNS in, out UNIQUE;

-- servers
DEFINE TABLE servers SCHEMAFULL;

DEFINE FIELD name ON TABLE servers TYPE string;
DEFINE FIELD icon ON TABLE servers TYPE string;
DEFINE FIELD banner ON TABLE servers TYPE string;
DEFINE FIELD channels ON TABLE servers TYPE array<record<channels>>;
DEFINE FIELD created_at ON TABLE channels TYPE datetime DEFAULT time::now();

-- channels
DEFINE TABLE channels SCHEMAFULL;

DEFINE FIELD name ON TABLE channels TYPE string;
DEFINE FIELD type ON TABLE channels TYPE string;
DEFINE FIELD private ON TABLE channels TYPE bool;
DEFINE FIELD created_at ON TABLE channels TYPE datetime DEFAULT time::now();

-- messages
DEFINE TABLE messages SCHEMAFULL;

DEFINE FIELD author ON TABLE channels TYPE record<users>;
DEFINE FIELD channel_id ON TABLE channels TYPE string;
DEFINE FIELD content ON TABLE channels TYPE string;
DEFINE FIELD edited ON TABLE channels TYPE bool;
DEFINE FIELD updated_at ON TABLE channels TYPE datetime DEFAULT time::now();
DEFINE FIELD created_at ON TABLE channels TYPE datetime DEFAULT time::now();

-- notifications
DEFINE TABLE notifications SCHEMALESS;

-- channel subscription
DEFINE TABLE subscribed TYPE RELATION FROM users TO channels;
DEFINE INDEX unique_relationships
        ON TABLE subscribed
        COLUMNS in, out UNIQUE;

-- server member
DEFINE TABLE member TYPE RELATION FROM users TO servers;
DEFINE INDEX unique_relationships
        ON TABLE member
        COLUMNS in, out UNIQUE;
