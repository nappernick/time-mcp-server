import { MCPClient } from '../src/mcp-client';
import * as path from 'path';
import { fileURLToPath } from 'url';

// ES-module equivalent of __dirname
const __filename = fileURLToPath(import.meta.url);
const __dirname  = path.dirname(__filename);

describe('MCPClient.parseNaturalTime – edge cases', () => {
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
      } catch { /* continue */ }
    }
    if (!serverPath) {
      throw new Error(
        'MCP server executable not found. Run: go build -o time-mcp-server .'
      );
    }

    client = new MCPClient(serverPath, [
      '--transport',      'stdio',
      '--local-timezone', 'UTC'
    ]);

    await client.connect();
    await client.initialize();
    await client.listTools();
  });

  afterAll(() => client?.disconnect());

  /* -------------------------------------------------------------- */
  it('throws on completely unparseable input', async () => {
    await expect(
      client.parseNaturalTime('gobbledygook 12345', 'UTC')
    ).rejects.toThrow();
  });

  it('parses "July 4, 2025" correctly in UTC', async () => {
    const result = await client.parseNaturalTime('July 4, 2025', 'UTC');

    expect(result).toBeDefined();
    expect(result.timezone).toBe('UTC');

    const dt = new Date(result.datetime);
    expect(dt.getUTCFullYear()).toBe(2025);
    expect(dt.getUTCMonth()).toBe(6); // July
    expect(dt.getUTCDate()).toBe(4);
  });

  it('handles ambiguous dates like "in 1 week"', async () => {
    const result = await client.parseNaturalTime('in 1 week', 'UTC');

    expect(result).toBeDefined();
    expect(result.timezone).toBe('UTC');

    const parsed = new Date(result.datetime);
    const now    = new Date();
    const diff   = (parsed.getTime() - now.getTime()) / (1000 * 60 * 60 * 24);

    expect(diff).toBeGreaterThan(5);
    expect(diff).toBeLessThan(10);
  });

  it('handles edge case with invalid timezone gracefully', async () => {
    await expect(
      client.parseNaturalTime('tomorrow at noon', 'Invalid/Timezone')
    ).rejects.toThrow();
  });

  it('parses relative times correctly', async () => {
    const result = await client.parseNaturalTime('in 2 hours', 'UTC');

    expect(result).toBeDefined();
    expect(result.timezone).toBe('UTC');

    const parsed = new Date(result.datetime);
    const now    = new Date();
    const diff   = (parsed.getTime() - now.getTime()) / (1000 * 60 * 60);

    expect(diff).toBeGreaterThan(1.5);
    expect(diff).toBeLessThan(2.5);
  });
});
