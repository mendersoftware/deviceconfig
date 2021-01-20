# coding: utf-8

"""
    Device Configure Internal API

    Internal API for managing persistent device connections. Intended for use by the web GUI.   # noqa: E501

    The version of the OpenAPI document: 1
    Contact: support@mender.io
    Generated by: https://openapi-generator.tech
"""


from __future__ import absolute_import

import unittest
import datetime

import internal_api
from internal_api.models.new_tenant import NewTenant  # noqa: E501
from internal_api.rest import ApiException

class TestNewTenant(unittest.TestCase):
    """NewTenant unit test stubs"""

    def setUp(self):
        pass

    def tearDown(self):
        pass

    def make_instance(self, include_optional):
        """Test NewTenant
            include_option is a boolean, when False only required
            params are included, when True both required and
            optional params are included """
        # model = internal_api.models.new_tenant.NewTenant()  # noqa: E501
        if include_optional :
            return NewTenant(
                tenant_id = '0'
            )
        else :
            return NewTenant(
                tenant_id = '0',
        )

    def testNewTenant(self):
        """Test NewTenant"""
        inst_req_only = self.make_instance(include_optional=False)
        inst_req_and_optional = self.make_instance(include_optional=True)


if __name__ == '__main__':
    unittest.main()
