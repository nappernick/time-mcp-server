// Global test type declarations

declare global {
    var testUtils: {
      originalConsoleLog: typeof console.log;
      originalConsoleError: typeof console.error;
    };
  }
  
  export {};