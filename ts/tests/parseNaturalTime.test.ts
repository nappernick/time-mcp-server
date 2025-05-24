import { MCPClient } from '../src/mcp-client';
import * as path from 'path';
import { fileURLToPath } from 'url';

// ES-module equivalent of __dirname
const __filename = fileURLToPath(import.meta.url);
const __dirname  = path.dirname(__filename);

describe('MCPClient.parseNaturalTime – happy path', () => {
  let client: MCPClient;
  let serverPath = '';

  /* -------------------------------------------------------------- */
  beforeAll(async () => {
    const tryPaths = [
      path.resolve(__dirname, '../time-mcp-server'),
      path.resolve(__dirname, '../../time-mcp-server'),
      path.resolve(__dirname, '../../../time-mcp-server')
    ];

    const fs = await import('fs');
    for (const p of tryPaths) {
      try {
        await fs.promises.access(p, fs.constants.F_OK);
        serverPath = p;
        break;
      } catch { /* keep searching */ }
    }
    if (!serverPath) {
      throw new Error(
        'MCP server executable not found. Run: go build -o time-mcp-server .'
      );
    }

    client = new MCPClient(serverPath, [
      '--transport',      'stdio',
      '--local-timezone', 'America/Chicago'
    ]);

    await client.connect();
    await client.initialize();
    await client.listTools();
  });

  afterAll(() => client?.disconnect());

  /* -------------------------------------------------------------- */
  it('parses "next Friday at noon" in Chicago time', async () => {
    const result = await client.parseNaturalTime(
      'next Friday at noon',
      'America/Chicago'
    );

    expect(result).toBeDefined();
    expect(result.timezone).toBe('America/Chicago');

    const dt = new Date(result.datetime);
    expect(dt.getDay()).toBe(5); // Friday

    const chicago = new Date(
      dt.toLocaleString('en-US', { timeZone: 'America/Chicago' })
    );
    expect(chicago.getHours()).toBe(12);
    expect(chicago.getMinutes()).toBe(0);
  });

  it('parses "in 3 days" in UTC', async () => {
    const result = await client.parseNaturalTime('in 3 days', 'UTC');

    expect(result).toBeDefined();
    expect(result.timezone).toBe('UTC');

    const parsed = new Date(result.datetime);
    const now    = new Date();
    const diff   = (parsed.getTime() - now.getTime()) / (1000 * 60 * 60 * 24);

    expect(diff).toBeGreaterThan(2.5);
    expect(diff).toBeLessThan(3.5);
  });

  it('parses "tomorrow at 9am" correctly', async () => {
    const result = await client.parseNaturalTime(
      'tomorrow at 9am',
      'America/Chicago'
    );

    expect(result).toBeDefined();
    expect(result.timezone).toBe('America/Chicago');

    const dt = new Date(result.datetime);
    const chicago = new Date(
      dt.toLocaleString('en-US', { timeZone: 'America/Chicago' })
    );
    expect(chicago.getHours()).toBe(9);
    expect(chicago.getMinutes()).toBe(0);
  });

  it('handles timezone conversion for parsed dates', async () => {
    const result = await client.parseNaturalTime('next Monday at 2pm', 'UTC');

    expect(result).toBeDefined();
    expect(result.timezone).toBe('UTC');

    const dt = new Date(result.datetime);
    expect(dt.getUTCHours()).toBe(14); // 2 pm
    expect(dt.getUTCDay()).toBe(1);    // Monday
  });
});
