import createClient from 'openapi-fetch';
import type { paths } from '../api-types';
import { notify } from './toast';

export const client = createClient<paths>({
  baseUrl: '/api',
});

// Middleware for global error handling
client.use({
  onResponse: async ({ response }) => {
    if (!response.ok) {
      try {
        const errorData = await response.clone().json();
        // If the backend returns a specific error message structure, handle it here
        const message = errorData.message || errorData.error || `API Error: ${response.statusText}`;
        notify.error(message);
      } catch {
        // Fallback for non-JSON errors
        notify.error(`API Error: ${response.statusText} (${response.status})`);
      }
    }
    return response;
  }
});
