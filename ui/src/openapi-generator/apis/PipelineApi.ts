/* tslint:disable */
/* eslint-disable */
/**
 * Enduro API
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * The version of the OpenAPI document: 
 * 
 *
 * NOTE: This class is auto generated by OpenAPI Generator (https://openapi-generator.tech).
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */


import * as runtime from '../runtime';
import {
    EnduroStoredPipelineResponse,
    EnduroStoredPipelineResponseFromJSON,
    EnduroStoredPipelineResponseToJSON,
    PipelineProcessingNotFoundResponseBody,
    PipelineProcessingNotFoundResponseBodyFromJSON,
    PipelineProcessingNotFoundResponseBodyToJSON,
    PipelineShowNotFoundResponseBody,
    PipelineShowNotFoundResponseBodyFromJSON,
    PipelineShowNotFoundResponseBodyToJSON,
    PipelineShowResponseBody,
    PipelineShowResponseBodyFromJSON,
    PipelineShowResponseBodyToJSON,
} from '../models';

export interface PipelineListRequest {
    name?: string;
}

export interface PipelineProcessingRequest {
    id: string;
}

export interface PipelineShowRequest {
    id: string;
}

/**
 * no description
 */
export class PipelineApi extends runtime.BaseAPI {

    /**
     * List all known pipelines
     * list pipeline
     */
    async pipelineListRaw(requestParameters: PipelineListRequest): Promise<runtime.ApiResponse<Array<EnduroStoredPipelineResponse>>> {
        const queryParameters: runtime.HTTPQuery = {};

        if (requestParameters.name !== undefined) {
            queryParameters['name'] = requestParameters.name;
        }

        const headerParameters: runtime.HTTPHeaders = {};

        const response = await this.request({
            path: `/pipeline`,
            method: 'GET',
            headers: headerParameters,
            query: queryParameters,
        });

        return new runtime.JSONApiResponse(response, (jsonValue) => jsonValue.map(EnduroStoredPipelineResponseFromJSON));
    }

    /**
     * List all known pipelines
     * list pipeline
     */
    async pipelineList(requestParameters: PipelineListRequest): Promise<Array<EnduroStoredPipelineResponse>> {
        const response = await this.pipelineListRaw(requestParameters);
        return await response.value();
    }

    /**
     * List all processing configurations of a pipeline given its ID
     * processing pipeline
     */
    async pipelineProcessingRaw(requestParameters: PipelineProcessingRequest): Promise<runtime.ApiResponse<Array<string>>> {
        if (requestParameters.id === null || requestParameters.id === undefined) {
            throw new runtime.RequiredError('id','Required parameter requestParameters.id was null or undefined when calling pipelineProcessing.');
        }

        const queryParameters: runtime.HTTPQuery = {};

        const headerParameters: runtime.HTTPHeaders = {};

        const response = await this.request({
            path: `/pipeline/{id}/processing`.replace(`{${"id"}}`, encodeURIComponent(String(requestParameters.id))),
            method: 'GET',
            headers: headerParameters,
            query: queryParameters,
        });

        return new runtime.JSONApiResponse<any>(response);
    }

    /**
     * List all processing configurations of a pipeline given its ID
     * processing pipeline
     */
    async pipelineProcessing(requestParameters: PipelineProcessingRequest): Promise<Array<string>> {
        const response = await this.pipelineProcessingRaw(requestParameters);
        return await response.value();
    }

    /**
     * Show pipeline by ID
     * show pipeline
     */
    async pipelineShowRaw(requestParameters: PipelineShowRequest): Promise<runtime.ApiResponse<PipelineShowResponseBody>> {
        if (requestParameters.id === null || requestParameters.id === undefined) {
            throw new runtime.RequiredError('id','Required parameter requestParameters.id was null or undefined when calling pipelineShow.');
        }

        const queryParameters: runtime.HTTPQuery = {};

        const headerParameters: runtime.HTTPHeaders = {};

        const response = await this.request({
            path: `/pipeline/{id}`.replace(`{${"id"}}`, encodeURIComponent(String(requestParameters.id))),
            method: 'GET',
            headers: headerParameters,
            query: queryParameters,
        });

        return new runtime.JSONApiResponse(response, (jsonValue) => PipelineShowResponseBodyFromJSON(jsonValue));
    }

    /**
     * Show pipeline by ID
     * show pipeline
     */
    async pipelineShow(requestParameters: PipelineShowRequest): Promise<PipelineShowResponseBody> {
        const response = await this.pipelineShowRaw(requestParameters);
        return await response.value();
    }

}
