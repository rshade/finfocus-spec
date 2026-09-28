import {
  Module,
  Controller,
  Inject,
  All,
  Req,
  Res,
  InternalServerErrorException,
  Optional,
  DynamicModule,
  Injectable,
} from '@nestjs/common';
import type { Request, Response } from 'express';
import { RESTGateway, RESTGatewayConfig } from 'finfocus-middleware';

export const REST_GATEWAY_OPTIONS = 'REST_GATEWAY_OPTIONS';

/**
 * NestJS Controller for FinFocus REST Gateway.
 * Handles all FinFocus REST API endpoints.
 */
@Controller()
export class FinFocusController {
  constructor(
    @Inject(REST_GATEWAY_OPTIONS) private gateway: RESTGateway
  ) {}

  @All('finfocus.v1.*path')
  async handleRequest(@Req() req: Request, @Res() res: Response) {
    try {
      await this.gateway.handleRequest(req, res);
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      throw new InternalServerErrorException(errorMessage);
    }
  }
}

/**
 * NestJS Module for FinFocus REST Gateway.
 * Provides automatic integration with NestJS applications.
 *
 * @example
 * ```typescript
 * import { Module } from '@nestjs/common';
 * import { FinFocusModule } from 'finfocus-framework-plugins';
 * import { CostSourceClient } from '@rshade/finfocus-client';
 *
 * @Module({
 *   imports: [
 *     FinFocusModule.register({
 *       costSourceClient: new CostSourceClient({
 *         baseUrl: 'https://plugin.example.com'
 *       })
 *     })
 *   ]
 * })
 * export class AppModule {}
 * ```
 */
@Module({})
export class FinFocusModule {
  static register(options: RESTGatewayConfig): DynamicModule {
    return {
      module: FinFocusModule,
      controllers: [FinFocusController],
      providers: [
        {
          provide: REST_GATEWAY_OPTIONS,
          useValue: new RESTGateway(options),
        },
        FinFocusGatewayService,
      ],
      exports: [FinFocusGatewayService],
    };
  }

  /**
   * Async registration for when client initialization is async.
   *
   * @example
   * ```typescript
   * FinFocusModule.registerAsync({
   *   useFactory: async (configService: ConfigService) => ({
   *     costSourceClient: new CostSourceClient({
   *       baseUrl: configService.get('PLUGIN_URL')
   *     })
   *   }),
   *   inject: [ConfigService]
   * })
   * ```
   */
  static registerAsync(options: {
    useFactory: (...args: any[]) => Promise<RESTGatewayConfig> | RESTGatewayConfig;
    inject?: any[];
  }): DynamicModule {
    return {
      module: FinFocusModule,
      controllers: [FinFocusController],
      providers: [
        {
          provide: REST_GATEWAY_OPTIONS,
          useFactory: async (...args: any[]) => {
            const config = await options.useFactory(...args);
            return new RESTGateway(config);
          },
          inject: options.inject || [],
        },
        FinFocusGatewayService,
      ],
      exports: [FinFocusGatewayService],
    };
  }
}

/**
 * Service class for accessing the gateway directly.
 * Useful for custom route handlers or middleware.
 *
 * @example
 * ```typescript
 * @Injectable()
 * export class MyService {
 *   constructor(private finFocusService: FinFocusGatewayService) {}
 *
 *   async handleCustomRequest(req: Request, res: Response) {
 *     return this.finFocusService.handleRequest(req, res);
 *   }
 * }
 * ```
 */
@Injectable()
export class FinFocusGatewayService {
  constructor(
    @Optional() @Inject(REST_GATEWAY_OPTIONS) private gateway?: RESTGateway
  ) {}

  async handleRequest(req: Request, res: Response): Promise<void> {
    if (!this.gateway) {
      throw new InternalServerErrorException('FinFocus gateway not configured');
    }
    await this.gateway.handleRequest(req, res);
  }
}
