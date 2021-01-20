# coding: utf-8

"""
    Device Configure Internal API

    Internal API for managing persistent device connections. Intended for use by the web GUI.   # noqa: E501

    The version of the OpenAPI document: 1
    Contact: support@mender.io
    Generated by: https://openapi-generator.tech
"""


import pprint
import re  # noqa: F401

import six

from internal_api.configuration import Configuration


class NewTenant(object):
    """NOTE: This class is auto generated by OpenAPI Generator.
    Ref: https://openapi-generator.tech

    Do not edit the class manually.
    """

    """
    Attributes:
      openapi_types (dict): The key is attribute name
                            and the value is attribute type.
      attribute_map (dict): The key is attribute name
                            and the value is json key in definition.
    """
    openapi_types = {
        'tenant_id': 'str'
    }

    attribute_map = {
        'tenant_id': 'tenant_id'
    }

    def __init__(self, tenant_id=None, local_vars_configuration=None):  # noqa: E501
        """NewTenant - a model defined in OpenAPI"""  # noqa: E501
        if local_vars_configuration is None:
            local_vars_configuration = Configuration()
        self.local_vars_configuration = local_vars_configuration

        self._tenant_id = None
        self.discriminator = None

        self.tenant_id = tenant_id

    @property
    def tenant_id(self):
        """Gets the tenant_id of this NewTenant.  # noqa: E501

        ID of new tenant.  # noqa: E501

        :return: The tenant_id of this NewTenant.  # noqa: E501
        :rtype: str
        """
        return self._tenant_id

    @tenant_id.setter
    def tenant_id(self, tenant_id):
        """Sets the tenant_id of this NewTenant.

        ID of new tenant.  # noqa: E501

        :param tenant_id: The tenant_id of this NewTenant.  # noqa: E501
        :type: str
        """
        if self.local_vars_configuration.client_side_validation and tenant_id is None:  # noqa: E501
            raise ValueError("Invalid value for `tenant_id`, must not be `None`")  # noqa: E501

        self._tenant_id = tenant_id

    def to_dict(self):
        """Returns the model properties as a dict"""
        result = {}

        for attr, _ in six.iteritems(self.openapi_types):
            value = getattr(self, attr)
            if isinstance(value, list):
                result[attr] = list(map(
                    lambda x: x.to_dict() if hasattr(x, "to_dict") else x,
                    value
                ))
            elif hasattr(value, "to_dict"):
                result[attr] = value.to_dict()
            elif isinstance(value, dict):
                result[attr] = dict(map(
                    lambda item: (item[0], item[1].to_dict())
                    if hasattr(item[1], "to_dict") else item,
                    value.items()
                ))
            else:
                result[attr] = value

        return result

    def to_str(self):
        """Returns the string representation of the model"""
        return pprint.pformat(self.to_dict())

    def __repr__(self):
        """For `print` and `pprint`"""
        return self.to_str()

    def __eq__(self, other):
        """Returns true if both objects are equal"""
        if not isinstance(other, NewTenant):
            return False

        return self.to_dict() == other.to_dict()

    def __ne__(self, other):
        """Returns true if both objects are not equal"""
        if not isinstance(other, NewTenant):
            return True

        return self.to_dict() != other.to_dict()
