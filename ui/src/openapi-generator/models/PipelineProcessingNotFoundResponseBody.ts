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

import { exists, mapValues } from '../runtime';
/**
 * Pipeline not found
 * @export
 * @interface PipelineProcessingNotFoundResponseBody
 */
export interface PipelineProcessingNotFoundResponseBody {
    /**
     * Identifier of missing pipeline
     * @type {string}
     * @memberof PipelineProcessingNotFoundResponseBody
     */
    id: string;
    /**
     * Message of error
     * @type {string}
     * @memberof PipelineProcessingNotFoundResponseBody
     */
    message: string;
}

export function PipelineProcessingNotFoundResponseBodyFromJSON(json: any): PipelineProcessingNotFoundResponseBody {
    return PipelineProcessingNotFoundResponseBodyFromJSONTyped(json, false);
}

export function PipelineProcessingNotFoundResponseBodyFromJSONTyped(json: any, ignoreDiscriminator: boolean): PipelineProcessingNotFoundResponseBody {
    if ((json === undefined) || (json === null)) {
        return json;
    }
    return {
        
        'id': json['id'],
        'message': json['message'],
    };
}

export function PipelineProcessingNotFoundResponseBodyToJSON(value?: PipelineProcessingNotFoundResponseBody | null): any {
    if (value === undefined) {
        return undefined;
    }
    if (value === null) {
        return null;
    }
    return {
        
        'id': value.id,
        'message': value.message,
    };
}


