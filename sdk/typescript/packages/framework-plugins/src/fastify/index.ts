import type { FastifyRequest, FastifyReply, FastifyInstance, FastifyPluginAsync } from 'fastify';
import { RESTGateway, RESTGatewayConfig } from 'finfocus-middleware';

/**
 * Creates a Fastify plugin for FinFocus REST Gateway.
 * Integrates the FinFocus REST gateway with Fastify applications.
 *
 * @param config - RESTGateway configuration with client instances
 * @returns Fastify plugin async function
 *
 * @example
 * ```typescript
 * import Fastify from 'fastify';
 * import { createFastifyPlugin } from 'finfocus-framework-plugins';
 * import { CostSourceClient } from '@rshade/finfocus-client';
 *
 * const fastify = Fastify();
 * const client = new CostSourceClient({ baseUrl: 'https://plugin.example.com' });
 * await fastify.register(createFastifyPlugin({ costSourceClient: client }));
 * await fastify.listen({ port: 3000 });
 * ```
 */
export function createFastifyPlugin(config: RESTGatewayConfig): FastifyPluginAsync {
  const gateway = new RESTGateway(config);

  return async (fastify: FastifyInstance) => {
    fastify.post('/*', async (request: FastifyRequest, reply: FastifyReply) => {
      // Fastify has already parsed the body, so dispatch directly instead of re-reading the stream.
      const result = await gateway.dispatch(request.url, request.body);
      return reply.status(result.status).send(result.body);
    });
  };
}

/**
 * Alternative route-based approach for Fastify.
 * More explicit control over the routing.
 *
 * @example
 * ```typescript
 * const fastify = Fastify();
 * await fastify.register(createFastifyRoutes, { config });
 * ```
 */
export const createFastifyRoutes: FastifyPluginAsync<{ config: RESTGatewayConfig }> = async (
  fastify: FastifyInstance,
  { config }
) => {
  const plugin = createFastifyPlugin(config);
  await plugin(fastify, {});
};
