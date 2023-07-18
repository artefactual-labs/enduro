/* tslint:disable */
/* eslint-disable */
/**
 * Enduro API
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * The version of the OpenAPI document: 1.0
 * 
 *
 * NOTE: This class is auto generated by OpenAPI Generator (https://openapi-generator.tech).
 * https://openapi-generator.tech
 * Do not edit the class manually.
 */

import { exists, mapValues } from '../runtime';
import type { EnduroStoredCollection } from './EnduroStoredCollection';
import {
    EnduroStoredCollectionFromJSON,
    EnduroStoredCollectionFromJSONTyped,
    EnduroStoredCollectionToJSON,
} from './EnduroStoredCollection';

/**
 * 
 * @export
 * @interface EnduroMonitorUpdate
 */
export interface EnduroMonitorUpdate {
    /**
     * Identifier of collection
     * @type {number}
     * @memberof EnduroMonitorUpdate
     */
    id: number;
    /**
     * 
     * @type {EnduroStoredCollection}
     * @memberof EnduroMonitorUpdate
     */
    item?: EnduroStoredCollection;
    /**
     * Type of the event
     * @type {string}
     * @memberof EnduroMonitorUpdate
     */
    type: string;
}

/**
 * Check if a given object implements the EnduroMonitorUpdate interface.
 */
export function instanceOfEnduroMonitorUpdate(value: object): boolean {
    let isInstance = true;
    isInstance = isInstance && "id" in value;
    isInstance = isInstance && "type" in value;

    return isInstance;
}

export function EnduroMonitorUpdateFromJSON(json: any): EnduroMonitorUpdate {
    return EnduroMonitorUpdateFromJSONTyped(json, false);
}

export function EnduroMonitorUpdateFromJSONTyped(json: any, ignoreDiscriminator: boolean): EnduroMonitorUpdate {
    if ((json === undefined) || (json === null)) {
        return json;
    }
    return {
        
        'id': json['id'],
        'item': !exists(json, 'item') ? undefined : EnduroStoredCollectionFromJSON(json['item']),
        'type': json['type'],
    };
}

export function EnduroMonitorUpdateToJSON(value?: EnduroMonitorUpdate | null): any {
    if (value === undefined) {
        return undefined;
    }
    if (value === null) {
        return null;
    }
    return {
        
        'id': value.id,
        'item': EnduroStoredCollectionToJSON(value.item),
        'type': value.type,
    };
}

