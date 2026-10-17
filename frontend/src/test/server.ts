import { setupServer } from 'msw/node';

/** MSW server for controller tests; handlers are added per test with server.use(). */
export const server = setupServer();
