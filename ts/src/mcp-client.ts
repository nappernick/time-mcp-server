/**
 * Minimal TypeScript MCP client that talks to the Go demo server.
 * Works under Node 18+ ESM (module:"nodenext").
 */

import { spawn, ChildProcess } from 'child_process';
import path from 'path';
import { fileURLToPath } from 'url';

/* ---------------------------------------------------------------------
 * SDK imports ― NOTE: no "/types" suffix; typings are at package root.
 * ------------------------------------------------------------------- */
import type {
  JSONRPCRequest,
  JSONRPCResponse,
  JSONRPCError,
  InitializeResult,
  ListToolsResult,
  CallToolResult,
  Tool,
  RequestId,
  ClientCapabilities,
  Implementation
} from '@modelcontextprotocol/sdk/types';

import {
  isJSONRPCResponse,
  isJSONRPCError,
  JSONRPC_VERSION,
  LATEST_PROTOCOL_VERSION,
  ErrorCode,
  McpError,
} from '@modelcontextprotocol/sdk/types';
import { parse } from 'tolerant-json-parser/lib';

/* ------------------------------------------------------------------ */
export const __filename = fileURLToPath(import.meta.url);
export const __dirname  = path.dirname(__filename);

/* ------------------------------------------------------------------ */
interface TimeResult {
  timezone: string;
  datetime: string;
  is_dst: boolean;
}

interface TimeConversionResult {
  source: TimeResult;
  target: TimeResult;
  time_difference: string;
}

interface PendingRequest {
  resolve: (v: JSONRPCResponse | JSONRPCError) => void;
  reject:  (e: Error) => void;
  timestamp: number;
  timeoutId?: NodeJS.Timeout; // To store the timeout ID
}

/* helper: pull out the text blob from CallToolResult --------------- */
function textOf(res: CallToolResult) {
  const t = res.content.find(
    (c: { type: string; text?: string }): c is { type: 'text'; text: string } =>
      c.type === 'text'
  );
  return t?.text ?? '';
}

/* ==================================================================
   MCPClient
   ================================================================= */
class MCPClient {
  private proc: ChildProcess | null = null;
  private nextId = 1;
  private pending = new Map<RequestId, PendingRequest>();
  private initialized = false;
  private toolList: Tool[] = [];
  private stderrOutput: string[] = []; // Buffer for stderr if needed for debugging

  // Define the stderr handler as a member variable to easily add/remove it
  private stderrHandler = (d: Buffer | string) => {
    const message = d.toString().trim();
    this.stderrOutput.push(message); // Buffer it
    // console.error('[server]', message); // Avoid direct console.error here during tests
  };


  constructor(private command: string, private args: string[] = []) {}

  /* ------------------------------------------------- connection --- */
  async connect(): Promise<void> {
    this.stderrOutput = []; // Clear buffer on new connection
    return new Promise((resolve, reject) => {
      this.proc = spawn(this.command, this.args, { stdio: ['pipe', 'pipe', 'pipe'] });

      if (!this.proc.stdin || !this.proc.stdout) {
        return reject(new McpError(ErrorCode.ConnectionClosed, 'failed to open pipes'));
      }

      this.proc.stdout.on('data', buf =>
        buf
          .toString()
          .split(/\r?\n/)
          .forEach((line: string) => line && this.handle(line))
      );

      // Attach the defined handler
      this.proc.stderr?.on('data', this.stderrHandler);

      this.proc.once('error', err => {
        reject(err);
      });
      
      this.proc.once('exit', (code, signal) => {
        // console.log(`[client] Server process exited with code ${code} and signal ${signal}`);
        // This might be too late or cause issues if disconnect is already called
      });

      setTimeout(() => (this.proc?.pid ? resolve() : reject(
        new McpError(ErrorCode.ConnectionClosed, 'process not running'))), 120);
    });
  }

  disconnect() {
    if (this.proc) {
      this.proc.stdout?.removeAllListeners();
      // Remove the specific stderr handler
      this.proc.stderr?.removeListener('data', this.stderrHandler);
      this.proc.removeAllListeners('error');
      this.proc.removeAllListeners('exit');

      if (this.proc.stdin && !this.proc.stdin.destroyed) {
        this.proc.stdin.end(); 
      }
      
      if (!this.proc.killed) {
        this.proc.kill(); 
      }
      this.proc = null;
    }
    
    this.pending.forEach(p => {
      if (p.timeoutId) clearTimeout(p.timeoutId); // Clear any pending timeouts
      p.reject(new McpError(ErrorCode.ConnectionClosed, 'client closed'));
    });
    this.pending.clear();
    this.initialized = false;

    // Optionally print buffered stderr if tests failed or for debugging
    // if (this.stderrOutput.length > 0 && process.env.JEST_WORKER_ID !== undefined) {
    //   console.log("Buffered stderr from server process:\n", this.stderrOutput.join("\n"));
    // }
  }

  /* -------------------------------------- low-level JSON-RPC ---- */
  private rpc<T>(method: string, params?: unknown): Promise<T> {
    if (!this.proc?.stdin || this.proc.stdin.destroyed) {
      return Promise.reject(new McpError(ErrorCode.ConnectionClosed, 'not connected or stdin closed'));
    }

    const id = this.nextId++;
    let safeParams: { [x: string]: unknown; _meta?: { [x: string]: unknown; progressToken?: string | number } } | undefined;
    if (params === undefined) {
      safeParams = undefined;
    } else if (typeof params === 'object' && params !== null && !Array.isArray(params)) {
      safeParams = params as { [x: string]: unknown };
    } else {
      return Promise.reject(new McpError(ErrorCode.InvalidRequest, 'params must be an object or undefined'));
    }
    
    const req: JSONRPCRequest = { jsonrpc: JSONRPC_VERSION, id, method, params: safeParams };
    let localTimeoutId: NodeJS.Timeout | undefined;

    return new Promise<T>((resolve, reject) => {
      const pendingRequest: PendingRequest = {
        resolve: (response: JSONRPCResponse | JSONRPCError) => {
          if (localTimeoutId) clearTimeout(localTimeoutId);
          if (isJSONRPCError(response)) {
            reject(new McpError(response.error.code, response.error.message, response.error.data));
          } else if (isJSONRPCResponse(response)) {
            resolve(response.result as T);
          } else {
            reject(new McpError(ErrorCode.InternalError, 'Invalid JSON-RPC response format'));
          }
        },
        reject: (err: Error) => {
          if (localTimeoutId) clearTimeout(localTimeoutId);
          reject(err);
        },
        timestamp: Date.now(),
      };
      this.pending.set(id, pendingRequest);
      
      try {
        if (this.proc?.stdin && !this.proc.stdin.destroyed) {
          this.proc.stdin.write(JSON.stringify(req) + '\n');
        } else {
          throw new McpError(ErrorCode.ConnectionClosed, 'stdin not available for writing');
        }
      } catch (error) {
        if (localTimeoutId) clearTimeout(localTimeoutId);
        this.pending.delete(id);
        reject(error); 
        return;
      }

      localTimeoutId = setTimeout(() => {
        if (this.pending.has(id)) {
          const timedOutReq = this.pending.get(id);
          this.pending.delete(id);
          // The reject function in timedOutReq will already clear its own timeoutId if it was set,
          // but it's good practice to ensure it's cleared here too.
          // No need to clear localTimeoutId again as it's this timeout.
          timedOutReq?.reject(new McpError(ErrorCode.RequestTimeout, `Request ${id} (${method}) timed out after 10s`));
        }
      }, 10_000);
      // Store the timeout ID on the pending request object itself, if needed elsewhere,
      // but localTimeoutId is sufficient for this promise.
      pendingRequest.timeoutId = localTimeoutId;
    });
  }

  private handle(line: string) {
    let resUnknown: unknown;
    try {
      resUnknown = parse(line); 
    } catch (e) {
      // console.error('[client] Failed to parse line as JSON:', line, e);
      return;
    }

    if (!isJSONRPCResponse(resUnknown) && !isJSONRPCError(resUnknown)) {
      // console.warn('[client] Received non-JSONRPC message:', resUnknown);
      return;
    }
    
    const res = resUnknown as JSONRPCResponse | JSONRPCError;

    if (res.id === null || res.id === undefined) { 
        // console.log('[client] Received notification or response without ID:', res);
        return;
    }
    
    const pending = this.pending.get(res.id);
    if (!pending) {
      // console.warn(`[client] Received response for unknown id ${res.id}:`, res);
      return;
    }

    this.pending.delete(res.id);
    // The resolve path in the promise will clear the timeout
    pending.resolve(res);
  }

  /* -------------------------------------- public API ------------ */
  async initialize(): Promise<InitializeResult> {
    const caps: ClientCapabilities = { tools: {}, resources: {}, prompts: {} };
    const clientInfo: Implementation = { name: 'ts-client', version: '1.0.0' };

    const res = await this.rpc<InitializeResult>('initialize', {
      protocolVersion: LATEST_PROTOCOL_VERSION,
      capabilities: caps,
      clientInfo
    });

    this.initialized = true;
    return res;
  }

  async listTools(): Promise<Tool[]> { 
    const result = await this.rpc<ListToolsResult>('tools/list');
    this.toolList = result.tools;
    return this.toolList;
  }

  async callTool(name: string, args: Record<string, unknown>): Promise<CallToolResult> {
    if (!this.initialized) {
        throw new McpError(ErrorCode.InternalError, 'Client not initialized. Call initialize() first.');
    }
    return this.rpc<CallToolResult>('tools/call', { name, arguments: args });
  }

  /* ---------------- helpers for Go demo server ------------------ */
  async getCurrentTime(tz?: string): Promise<TimeResult> {
    const res = await this.callTool('get_current_time', tz ? { timezone: tz } : {});
    const toolOutputText = textOf(res);
    try {
      return parse(toolOutputText) as TimeResult;
    } catch (e: any) {
      throw new McpError(ErrorCode.InternalError, `Tool 'get_current_time' returned non-JSON output: ${toolOutputText}`, e.message);
    }
  }

  async convertTime(
    src: string,
    time: string,
    dst: string
  ): Promise<TimeConversionResult> {
    const res = await this.callTool('convert_time', {
      source_timezone: src,
      time,
      target_timezone: dst
    });
    const toolOutputText = textOf(res);
    try {
      return parse(toolOutputText) as TimeConversionResult;
    } catch (e: any) {
      throw new McpError(ErrorCode.InternalError, `Tool 'convert_time' returned non-JSON output: ${toolOutputText}`, e.message);
    }
  }

  async parseNaturalTime(expr: string, tz?: string): Promise<TimeResult> {
    const callToolResponse = await this.callTool(
      'parse_natural_time',
      tz ? { expression: expr, timezone: tz } : { expression: expr }
    );
    const toolOutputText = textOf(callToolResponse);
  
    try {
      const parsedResult = parse(toolOutputText); 
      
      if (typeof parsedResult === 'object' && parsedResult !== null && 
          'datetime' in parsedResult && 'timezone' in parsedResult && 'is_dst' in parsedResult) {
          return parsedResult as TimeResult;
      }
      throw new McpError(ErrorCode.InternalError, `Tool returned unexpected JSON structure: ${toolOutputText}`);
    } catch (e: any) {
      if (e instanceof McpError) throw e; 
      
      throw new McpError(ErrorCode.InternalError, toolOutputText, e.message);
    }
  }
}

/* ==================================================================
   Exports
   ================================================================= */
export { MCPClient };
export type { TimeResult, TimeConversionResult };

/* ==================================================================
   CLI (optional)  →  tsx mcp-client.ts
   ================================================================= */
async function main() {
  const mode = process.argv[2] ?? '';
  if (mode === 'interactive') {
    console.log('interactive mode not implemented');
    return;
  }

  const srvPath = path.resolve(__dirname, '../time-mcp-server'); 
  if (!require('fs').existsSync(srvPath)) { 
      console.error(`Server executable not found at ${srvPath}. Run: go build -o time-mcp-server . in the server directory.`);
      process.exit(1);
  }
  
  const cli = new MCPClient(srvPath, ['--transport', 'stdio', '--local-timezone', 'UTC']);

  try {
    await cli.connect();
    await cli.initialize();
    await cli.listTools();

    const res = await cli.parseNaturalTime('next Friday at noon', 'America/Chicago');
    console.log(res);
  } catch (error) {
    console.error('CLI Error:', error);
  } finally {
    cli.disconnect();
  }
}

if (import.meta.url.startsWith('file://') && process.argv[1] === fileURLToPath(import.meta.url)) {
  main().catch(e => {
    process.exit(1);
  });
}