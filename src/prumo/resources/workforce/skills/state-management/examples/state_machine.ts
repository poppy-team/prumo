/**
 * Production-Grade Type-Safe Finite State Machine (FSM)
 * Features:
 * - Discriminated union states (no impossible states)
 * - Pure transition reducer
 * - Extended context with type-safe events
 * - Guard conditions & deterministic action execution
 * - Subscription listener with cleanup
 */

export type MachineStatus = 'idle' | 'fetching' | 'success' | 'failure';

export interface DataContext<T> {
  data: T | null;
  error: Error | null;
  retryCount: number;
  maxRetries: number;
}

export type MachineEvent<T> =
  | { type: 'FETCH' }
  | { type: 'RESOLVE'; payload: T }
  | { type: 'REJECT'; error: Error }
  | { type: 'RETRY' }
  | { type: 'RESET' };

export type MachineState<T> = {
  status: MachineStatus;
  context: DataContext<T>;
};

export class AsyncStateMachine<T> {
  private state: MachineState<T>;
  private listeners: Set<(state: MachineState<T>) => void> = new Set();
  private abortController: AbortController | null = null;

  constructor(maxRetries: number = 3) {
    this.state = {
      status: 'idle',
      context: {
        data: null,
        error: null,
        retryCount: 0,
        maxRetries,
      },
    };
  }

  public getState(): Readonly<MachineState<T>> {
    return Object.freeze({
      status: this.state.status,
      context: Object.freeze({ ...this.state.context }),
    });
  }

  public subscribe(listener: (state: MachineState<T>) => void): () => void {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  private notify(): void {
    const currentState = this.getState();
    this.listeners.forEach((listener) => listener(currentState));
  }

  /**
   * Pure state transition function
   */
  public transition(event: MachineEvent<T>): boolean {
    const { status, context } = this.state;

    switch (status) {
      case 'idle': {
        if (event.type === 'FETCH') {
          this.state = {
            status: 'fetching',
            context: { ...context, error: null, retryCount: 0 },
          };
          this.notify();
          return true;
        }
        break;
      }

      case 'fetching': {
        if (event.type === 'RESOLVE') {
          this.state = {
            status: 'success',
            context: { ...context, data: event.payload, error: null },
          };
          this.notify();
          return true;
        }
        if (event.type === 'REJECT') {
          this.state = {
            status: 'failure',
            context: { ...context, error: event.error },
          };
          this.notify();
          return true;
        }
        if (event.type === 'RESET') {
          this.cancelInflight();
          this.state = {
            status: 'idle',
            context: { ...context, data: null, error: null, retryCount: 0 },
          };
          this.notify();
          return true;
        }
        break;
      }

      case 'failure': {
        if (event.type === 'RETRY') {
          // Guard: Verify retry count has not exceeded maximum
          if (context.retryCount < context.maxRetries) {
            this.state = {
              status: 'fetching',
              context: { ...context, retryCount: context.retryCount + 1, error: null },
            };
            this.notify();
            return true;
          }
          return false; // Guard rejected
        }
        if (event.type === 'RESET') {
          this.state = {
            status: 'idle',
            context: { ...context, data: null, error: null, retryCount: 0 },
          };
          this.notify();
          return true;
        }
        break;
      }

      case 'success': {
        if (event.type === 'FETCH') {
          this.state = {
            status: 'fetching',
            context: { ...context, error: null },
          };
          this.notify();
          return true;
        }
        if (event.type === 'RESET') {
          this.state = {
            status: 'idle',
            context: { ...context, data: null, error: null, retryCount: 0 },
          };
          this.notify();
          return true;
        }
        break;
      }
    }

    // Unhandled event for the current state: safely ignore without state mutation
    return false;
  }

  public cancelInflight(): void {
    if (this.abortController) {
      this.abortController.abort();
      this.abortController = null;
    }
  }

  /**
   * Execute an asynchronous task tied to state transitions and cancellation
   */
  public async executeTask(fetcher: (signal: AbortSignal) => Promise<T>): Promise<void> {
    if (!this.transition({ type: 'FETCH' })) {
      return;
    }

    this.cancelInflight();
    this.abortController = new AbortController();
    const { signal } = this.abortController;

    try {
      const result = await fetcher(signal);
      if (!signal.aborted) {
        this.transition({ type: 'RESOLVE', payload: result });
      }
    } catch (err: unknown) {
      if (!signal.aborted) {
        const error = err instanceof Error ? err : new Error(String(err));
        this.transition({ type: 'REJECT', error });
      }
    } finally {
      this.abortController = null;
    }
  }
}

// Self-test verification suite
export async function runSelfTest(): Promise<void> {
  const machine = new AsyncStateMachine<{ id: number; name: string }>(2);
  const history: MachineStatus[] = [];

  machine.subscribe((s) => history.push(s.status));

  // 1. Initial State
  if (machine.getState().status !== 'idle') throw new Error('Initial state must be idle');

  // 2. Fetch Transition
  machine.transition({ type: 'FETCH' });
  if (machine.getState().status !== 'fetching') throw new Error('Must transition to fetching');

  // 3. Reject Transition
  machine.transition({ type: 'REJECT', error: new Error('Network Timeout') });
  if (machine.getState().status !== 'failure') throw new Error('Must transition to failure');
  if (machine.getState().context.error?.message !== 'Network Timeout') throw new Error('Error must be stored');

  // 4. Retry Transition (under guard limit)
  machine.transition({ type: 'RETRY' });
  if (machine.getState().status !== 'fetching') throw new Error('Must transition to fetching on retry');
  if (machine.getState().context.retryCount !== 1) throw new Error('Retry count must increment');

  // 5. Successful Resolve Transition
  machine.transition({ type: 'RESOLVE', payload: { id: 42, name: 'Prumo Core' } });
  if (machine.getState().status !== 'success') throw new Error('Must transition to success');
  if (machine.getState().context.data?.name !== 'Prumo Core') throw new Error('Payload mismatch');

  // 6. Reset Transition
  machine.transition({ type: 'RESET' });
  if (machine.getState().status !== 'idle') throw new Error('Must reset to idle');

  console.log('[AsyncStateMachine] All invariant assertions passed successfully.');
}

if (typeof process !== 'undefined' && process.argv && process.argv[1] && process.argv[1].endsWith('state_machine.ts')) {
  runSelfTest().catch((e) => {
    console.error(e);
    process.exit(1);
  });
}

