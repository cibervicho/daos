#!/usr/bin/python3
"""
(C) Copyright 2021 Intel Corporation.

SPDX-License-Identifier: BSD-2-Clause-Patent
"""
from concurrent.futures import ThreadPoolExecutor, as_completed, TimeoutError
from logging import getLogger

from avocado.utils.process import CmdResult


class ThreadResult():
    """Class containing the results of a method executed by the ThreadManager class."""

    def __init__(self, id, passed, args, result):
        """Initialize a ThreadResult object.

        Args:
            id (int): the thread id for this result
            passed (bool): whether the thread completed or raised an exception
            args (dict): the arguments passed to the thread method
            result (object): the object returned by the thread method
        """
        self.id = id
        self.passed = passed
        self.args = args
        self.result = result

    def __str__(self):
        """Return the string respresentation of this object.

        Returns:
            str: the string respresentation of this object

        """
        info = ["Thread {} results:".format(self.id), "  args: {}".format(self.args), "  result:"]
        if isinstance(self.result, CmdResult):
            info.append("    command:     {}".format(self.result.command))
            info.append("    exit_status: {}".format(self.result.exit_status))
            info.append("    duration:    {}".format(self.result.duration))
            info.append("    interrupted: {}".format(self.result.interrupted))
            info.append("    stdout:")
            for line in self.result.stdout_text.splitlines():
                info.append("      {}".format(line))
            info.append("    stderr:")
            for line in self.result.stderr_text.splitlines():
                info.append("      {}".format(line))
        else:
            for line in str(self.result).splitlines():
                info.append("    {}".format(line))
        return "\n".join(info)


class ThreadManager():
    """Class to manage running any method as multiple threads."""

    def __init__(self, method, timeout=None):
        """Initialize a ThreadManager object with the the method to run as a thread.

        Args:
            method (callable): [description]
            timeout (int, optional): [description]. Defaults to None.
        """
        self.log = getLogger()
        self.method = method
        self.timeout = timeout
        self.job_kwargs = []
        self.futures = {}

    @property
    def qty(self):
        """Get the number of threads.

        Returns:
            int: number of threads

        """
        return len(self.job_kwargs)

    def add(self, **kwargs):
        """Add a thread to run by specifying the keyword arguments for the thread method."""
        self.job_kwargs.append(kwargs)

    def run(self):
        """Asynchronously run the method as a thread for each set of method arguments.

        Returns:
            list: a list of ThreadResults for the execution of each method.

        """
        results = []
        with ThreadPoolExecutor() as thread_executor:
            self.log.info("Submitting %d threads ...", len(self.job_kwargs))
            futures = {
                thread_executor.submit(self.method, **kwargs): index
                for index, kwargs in enumerate(self.job_kwargs)}
            try:
                for future in as_completed(futures, self.timeout):
                    id = futures[future]
                    try:
                        results.append(
                            ThreadResult(id, True, self.job_kwargs[id], future.result()))
                    except Exception as error:
                        results.append(ThreadResult(id, False, self.job_kwargs[id], str(error)))
            except TimeoutError as error:
                for future in futures:
                    if not future.done():
                        results.append(ThreadResult(id, False, self.job_kwargs[id], str(error)))
        return results

    def check(self, results):
        """Display the results from self.run() and indicate if any threads failed.

        Args:
            results (list): a list of ThreadResults from self.run()

        Returns:
            bool: True if all threads passed; false otherwise.

        """
        failed = []
        self.log.info("Results from threads that passed:")
        for result in results:
            if result.passed:
                self.log.info(str(result))
            else:
                failed.append(result)
        if failed:
            self.log.info("Results from threads that failed:")
            for result in failed:
                self.log.info(str(result))
        return len(failed) > 0

    def check_run(self):
        """Run the threads and check thr result.

        Returns:
            bool: True if all threads passed; false otherwise.

        """
        return self.check(self.run())
