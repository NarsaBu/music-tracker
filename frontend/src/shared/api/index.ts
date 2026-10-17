import { FetchHttpClient } from './HttpClient';

export { ApiContractError, ApiError, type ApiFieldError } from './ApiError';
export { FetchHttpClient, type HttpClient } from './HttpClient';

/** Application-wide client. Same origin: Vite (dev) and nginx (prod) proxy to the backend. */
export const httpClient = new FetchHttpClient();
