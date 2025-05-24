// Test-wide setup – runs before all test files.

/* ---------------------------------------------------------------- *
 * Polyfill __filename / __dirname for ESM test files -------------- */
 import { fileURLToPath } from 'url';
 import path from 'path';
 
 (globalThis as any).__filename = fileURLToPath(import.meta.url);
 (globalThis as any).__dirname  = path.dirname((globalThis as any).__filename);
 /* ---------------------------------------------------------------- */
 
 // Note: jest.setTimeout() is configured in jest.config.js instead of here
 // to avoid "jest is not defined" errors in ESM setup files
 
 /* -------- Optional console silencing for cleaner test output ----- */
 const originalConsoleLog   = console.log;
 const originalConsoleError = console.error;
 const isVerbose            = process.env.VERBOSE_TESTS === 'true';
 
 if (!isVerbose) {
   console.log = (...args: unknown[]) => {
     const msg = args.join(' ');
     if (
       !msg.includes('🔌') &&
       !msg.includes('📋') &&
       !msg.includes('→')  &&
       !msg.includes('←')
     ) {
       originalConsoleLog(...args);
     }
   };
 }
 
 // Global test utilities
 (global as any).testUtils = { originalConsoleLog, originalConsoleError };
 
 afterAll(() => {
   console.log  = originalConsoleLog;
   console.error = originalConsoleError;
 });