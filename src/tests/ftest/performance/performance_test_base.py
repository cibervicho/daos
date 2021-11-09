#!/usr/bin/python3
"""
  (C) Copyright 2018-2021 Intel Corporation.

  SPDX-License-Identifier: BSD-2-Clause-Patent
"""
from ior_test_base import IorTestBase
from mdtest_test_base import MdtestBase

class PerformanceTestBase(IorTestBase, MdtestBase):
    # pylint: disable=too-many-ancestors
    # pylint: disable=too-few-public-methods
    """Base performance class."""

    def print_performance_params(self, cmd):
        """Print performance parameters.

        Args:
            cmd (str): ior or mdtest

        """
        # Start with common parameters
        # Build a list of [PARAM_NAME, PARAM_VALUE]
        params = [
            ["TEST_NAME", self.test_id],
            ["NUM_SERVERS", len(self.hostlist_servers)],
            ["NUM_CLIENTS", len(self.hostlist_clients)],
            ["PPC", self.processes / int(len(self.hostlist_clients))],
            ["PPN", self.processes / int(len(self.hostlist_clients))]
        ]

        # Get ior/mdtest specific parameters
        cmd = cmd.lower()
        if cmd == "ior":
            params += [
                ["OCLASS", self.ior_cmd.dfs_oclass.value],
                ["XFER_SIZE", self.ior_cmd.transfer_size.value],
                ["BLOCK_SIZE", self.ior_cmd.block_size.value],
                ["SW_TIME", self.ior_cmd.sw_deadline.value],
                ["CHUNK_SIZE", self.ior_cmd.dfs_chunk.value]
            ]
        elif cmd == "mdtest":
            params += [
                ["OCLASS", self.mdtest_cmd.dfs_oclass.value],
                ["DIR_OCLASS", self.mdtest_cmd.transfer_size.value],
                ["SW_TIME", self.mdtest_cmd.stonewall_timer.value],
                ["CHUNK_SIZE", self.mdtest_cmd.dfs_chunk.value]
            ]
        else:
            self.fail("Invalid cmd: {}".format(cmd))

        # Print and align all parameters in the format:
        # PARAM_NAME : PARAM_VALUE
        self.log.info("PERFORMANCE PARAMS START")
        max_len = max([len(param[0]) for param in params])
        for param in params:
            self.log.info("{:<{}} : {}".format(param[0], max_len, param[1]))
        self.log.info("PERFORMANCE PARAMS END")

    def print_system_status(self):
        """TODO"""
        pass

    def run_performance_ior(self, write_flags=None, read_flags=None):
        """Run an IOR performance test.

        Args:
            write_flags (str, optional): IOR flags for write phase.
                Defaults to ior/write_flags in the config.
            read_flags (str, optional): IOR flags for read phase.
                Defaults to ior/read_flags in the config.

        """
        if write_flags is None:
            write_flags = self.params.get("write_flags", "/run/ior/*")
        if read_flags is None:
            read_flags = self.params.get("read_flags", "/run/ior/*")

        self.print_performance_params("ior")

        # TODO for debugging
        if self.ior_cmd.dfs_oclass.value == "EC_16P2GX":
            self.fail("Need more nodes")

        self.log.info("Running IOR write")
        self.ior_cmd.flags.update(write_flags)
        self.run_ior_with_pool()

        self.log.info("Running IOR read")
        self.ior_cmd.flags.update(read_flags)
        self.ior_cmd.sw_wearout.update(None)
        self.ior_cmd.sw_deadline.update(None)
        self.run_ior_with_pool(create_cont=False)

    def run_performance_ior_easy(self):
        """Run an IOR easy performance test."""
        write_flags = self.params.get(
            "write_flags", "/run/ior/*", "-w -C -e -g -G 27 -k -Q 1 -v")
        read_flags = self.params.get(
            "read_flags", "/run/ior/*", "-r -R -C -e -g -G 27 -k -Q 1 -v")
        self.run_performance_ior(write_flags, read_flags)

    def run_performance_mdtest(self, flags=None):
        """Run an MdTest performance test.

        Args:
            flags (str, optional): MdTest flags.
                Defaults to mdtest/flags in the config.

        """
        if flags is None:
            flags = self.params.get("flags", "/run/mdtest/*")
        self.mdtest_cmd.flags.update(flags)
        self.print_performance_params()

        # TODO for debugging
        if self.mdtest_cmd.dfs_oclass.value == "EC_16P2GX":
            self.fail("Need more nodes")

        self.log.info("Running MDTEST")
        self.execute_mdtest()