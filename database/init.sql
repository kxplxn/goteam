CREATE TABLE IF NOT EXISTS app."user" (
  id       VARCHAR(15) PRIMARY KEY,
  password BYTEA       NOT NULL
);

CREATE TABLE IF NOT EXISTS app.user_board (
  id      SERIAL      PRIMARY KEY,
  userID  VARCHAR(15) NOT NULL    REFERENCES app."user",
  boardID SERIAL      NOT NULL    REFERENCES app.board,
  isAdmin BOOLEAN     NOT NULL
);

CREATE TABLE IF NOT EXISTS app.board (
  id   SERIAL      PRIMARY KEY,
  name VARCHAR(30) NOT NULL
);

CREATE TABLE IF NOT EXISTS app."column" (
  id      SERIAL   PRIMARY KEY,
  tableID SERIAL   NOT NULL    REFERENCES app.board,
  "order" SMALLINT NOT NULL
);

CREATE TABLE IF NOT EXISTS app.task (
  id          SERIAL      PRIMARY KEY,
  columnID    SERIAL      NOT NULL    REFERENCES app."column",
  title       VARCHAR(50) NOT NULL,
  description TEXT
);

CREATE TABLE IF NOT EXISTS app.subtask (
  id     SERIAL      PRIMARY KEY,
  taskID SERIAL      NOT NULL    REFERENCES app.task,
  title  VARCHAR(50) NOT NULL,
  isDone BOOLEAN     NOT NULL
);
