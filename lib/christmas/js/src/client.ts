import { LEDClientMessage, LEDServerMessage } from "./christmaspb/christmas.js";
import WebSocket from "isomorphic-ws";

// SessionEvents contains events emitted by Session.
export interface SessionEvents {
  open: CustomEvent<void>;
  close: CustomEvent<void>;
  message: CustomEvent<LEDServerMessage>;
}

interface ClientEventTarget extends EventTarget {
  addEventListener<K extends keyof SessionEvents>(
    type: K,
    listener: (ev: SessionEvents[K]) => void,
    options?: boolean | AddEventListenerOptions,
  ): void;
  addEventListener(
    type: string,
    callback: EventListenerOrEventListenerObject | null,
    options?: EventListenerOptions | boolean,
  ): void;
  removeEventListener<K extends keyof SessionEvents>(
    type: K,
    listener: (ev: SessionEvents[K]) => void,
    options?: boolean | EventListenerOptions,
  ): void;
  removeEventListener(
    type: string,
    callback: EventListenerOrEventListenerObject | null,
    options?: EventListenerOptions | boolean,
  ): void;
}

const clientEventTarget = EventTarget as {
  new (): ClientEventTarget;
  prototype: ClientEventTarget;
};

export class Client extends clientEventTarget {
  private ws: WebSocket | null;
  private url: string;
  private openPromise: Promise<void> = Promise.resolve();

  constructor(public readonly address: string) {
    super();
    console.log("connecting to", address);

    this.ws = null;
    this.url = `ws://${address}/ws`;
  }

  // connect connects to the server and authenticates with the given secret.
  // It blocks until the connection is established.
  async connect(secret: string) {
    const authedEventPromise = this.nextMessage();

    this.init();
    await this.openPromise;

    this.send({ authenticate: { secret } });

    const authedEvent = await authedEventPromise;
    if (!authedEvent.authenticate?.success) {
      throw new Error("authentication failed");
    }
  }

  close(graceful: boolean = true) {
    if (this.ws) this.ws.close(graceful ? 1000 : 1001);
  }

  send(message: LEDClientMessage) {
    if (!this.ws) throw new Error("websocket is closed");
    this.ws.send(LEDClientMessage.encode(message).finish());
  }

  async nextMessage(type?: keyof LEDClientMessage): Promise<LEDServerMessage> {
    return new Promise((resolve, reject) => {
      const onMessage = (ev: CustomEvent<LEDServerMessage> | null) => {
        if (!ev) {
          unbind();
          reject(new Error("session closed"));
          return;
        }

        if (type && !ev?.detail[type]) {
          return;
        }

        unbind();
        resolve(ev.detail);
      };

      const onClose = () => onMessage(null);

      const unbind = () => {
        this.removeEventListener("message", onMessage);
        this.removeEventListener("close", onClose);
      };

      this.addEventListener("message", onMessage);
      this.addEventListener("close", onClose);
    });
  }

  async *messages(): AsyncGenerator<LEDServerMessage> {
    while (true) {
      yield await this.nextMessage();
    }
  }

  private init() {
    if (this.ws) return;

    console.log("connecting to websocket...");

    const ws = new WebSocket(this.url);
    this.ws = ws;
    this.ws.addEventListener("open", this.onOpen.bind(this));
    this.ws.addEventListener("close", this.onClose.bind(this));
    this.ws.addEventListener("message", this.onMessage.bind(this));

    this.openPromise = new Promise((resolve, reject) => {
      if (!ws) {
        throw "ws should not be null";
      }
      ws.addEventListener("open", () => {
        resolve();
      });
      ws.addEventListener("close", (ev) =>
        reject(new Error(`closed (code ${ev.code}): ${ev.reason}`)),
      );
      ws.addEventListener("error", (ev) => {
        reject(new Error(`websocket connection error: server unreachable`));
      });
    });
  }

  private async onOpen() {
    this.dispatchEvent(new CustomEvent("open"));
  }

  private async onClose() {
    this.dispatchEvent(new CustomEvent("close"));
    this.ws = null;
    // this.init(); // immediately reconnect
  }

  private async onMessage(event: MessageEvent) {
    const data = LEDServerMessage.decode(new Uint8Array(event.data));
    this.dispatchEvent(new CustomEvent("message", { detail: data }));
  }
}
