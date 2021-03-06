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
 * 
 * @export
 * @interface CollectionBulkRequestBody
 */
export interface CollectionBulkRequestBody {
    /**
     * 
     * @type {string}
     * @memberof CollectionBulkRequestBody
     */
    operation: CollectionBulkRequestBodyOperationEnum;
    /**
     * 
     * @type {number}
     * @memberof CollectionBulkRequestBody
     */
    size?: number;
    /**
     * 
     * @type {string}
     * @memberof CollectionBulkRequestBody
     */
    status: CollectionBulkRequestBodyStatusEnum;
}

export function CollectionBulkRequestBodyFromJSON(json: any): CollectionBulkRequestBody {
    return CollectionBulkRequestBodyFromJSONTyped(json, false);
}

export function CollectionBulkRequestBodyFromJSONTyped(json: any, ignoreDiscriminator: boolean): CollectionBulkRequestBody {
    if ((json === undefined) || (json === null)) {
        return json;
    }
    return {
        
        'operation': json['operation'],
        'size': !exists(json, 'size') ? undefined : json['size'],
        'status': json['status'],
    };
}

export function CollectionBulkRequestBodyToJSON(value?: CollectionBulkRequestBody | null): any {
    if (value === undefined) {
        return undefined;
    }
    if (value === null) {
        return null;
    }
    return {
        
        'operation': value.operation,
        'size': value.size,
        'status': value.status,
    };
}

/**
* @export
* @enum {string}
*/
export enum CollectionBulkRequestBodyOperationEnum {
    Retry = 'retry',
    Cancel = 'cancel',
    Abandon = 'abandon'
}
/**
* @export
* @enum {string}
*/
export enum CollectionBulkRequestBodyStatusEnum {
    New = 'new',
    InProgress = 'in progress',
    Done = 'done',
    Error = 'error',
    Unknown = 'unknown',
    Queued = 'queued',
    Pending = 'pending',
    Abandoned = 'abandoned'
}


